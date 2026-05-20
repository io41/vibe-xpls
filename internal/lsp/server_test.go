package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/io41/vibe-xpls/internal/analyzer"
	"github.com/io41/vibe-xpls/internal/source"
	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestInitializeAdvertisesCapabilitiesAndNegotiatesPositionEncoding(t *testing.T) {
	tests := []struct {
		name    string
		offered []string
		want    string
	}{
		{name: "utf8 preferred", offered: []string{"utf-16", "utf-8"}, want: "utf-8"},
		{name: "fallback", offered: []string{"utf-16"}, want: "utf-16"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := runServerFrames(t, requestFrame(t, 1, "initialize", map[string]any{
				"rootUri": fileURI(testRoot(t)),
				"capabilities": map[string]any{
					"general": map[string]any{"positionEncodings": tt.offered},
				},
			}))

			result := resultMap(t, responseForID(t, messages, 1))
			capabilities := asMap(t, result["capabilities"])
			if capabilities["positionEncoding"] != tt.want {
				t.Fatalf("positionEncoding = %v, want %q", capabilities["positionEncoding"], tt.want)
			}
			if capabilities["hoverProvider"] != true {
				t.Fatalf("hoverProvider = %v, want true", capabilities["hoverProvider"])
			}
			if capabilities["textDocumentSync"] != float64(1) {
				t.Fatalf("textDocumentSync = %v, want full sync", capabilities["textDocumentSync"])
			}
			completion := asMap(t, capabilities["completionProvider"])
			triggers := asSlice(t, completion["triggerCharacters"])
			if containsString(triggers, ".") ||
				containsString(triggers, ":") ||
				!containsString(triggers, "\n") {
				t.Fatalf("triggerCharacters = %#v, want newline key-context trigger without dot or colon", completion["triggerCharacters"])
			}
			info := asMap(t, result["serverInfo"])
			if info["name"] != "vibe-xpls" || info["version"] != "v0.0.1" {
				t.Fatalf("serverInfo = %#v", info)
			}
		})
	}
}

func TestShutdownReturnsNullResult(t *testing.T) {
	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(testRoot(t)), "capabilities": map[string]any{}}),
		requestFrame(t, 2, "shutdown", nil),
	)

	response := responseForID(t, messages, 2)
	if string(response.Result) != "null" {
		t.Fatalf("shutdown result = %q, want null", string(response.Result))
	}
}

func TestUnknownRequestReturnsMethodNotFoundAndUnknownNotificationIsIgnored(t *testing.T) {
	messages := runServerFrames(t,
		requestFrame(t, 9, "workspace/nope", nil),
		notificationFrame(t, "workspace/nope", nil),
	)

	if len(messages) != 1 {
		t.Fatalf("messages = %d, want only response to unknown request: %#v", len(messages), messages)
	}
	response := responseForID(t, messages, 9)
	if response.Error == nil || response.Error.Code != methodNotFound {
		t.Fatalf("error = %#v, want method not found", response.Error)
	}
}

func TestUninitializedDocumentOperationsDoNotPanic(t *testing.T) {
	uri := fileURI(filepath.Join(testRoot(t), "api", "composition.yaml"))
	messages := runServerFrames(t,
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": "kind: Composition\n"},
		}),
		notificationFrame(t, "textDocument/didChange", map[string]any{
			"textDocument":   map[string]any{"uri": uri},
			"contentChanges": []map[string]any{{"text": "kind: Composition\n"}},
		}),
		requestFrame(t, 1, "textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 0},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 0},
		}),
		notificationFrame(t, "textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		}),
	)

	if string(responseForID(t, messages, 1).Result) != "null" {
		t.Fatalf("uninitialized hover result = %q, want null", string(responseForID(t, messages, 1).Result))
	}
	completion := resultMap(t, responseForID(t, messages, 2))
	if got := len(asSlice(t, completion["items"])); got != 0 {
		t.Fatalf("uninitialized completion items = %d, want 0", got)
	}
	diagnostics := diagnosticsFromLastNotification(t, messages)
	if got := len(diagnostics); got != 0 {
		t.Fatalf("close diagnostics = %d, want empty clear", got)
	}
}

func TestDidClosePublishesEmptyDiagnostics(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "composition.yaml"))
	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": map[string]any{}}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":  uri,
				"text": "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n",
			},
		}),
		notificationFrame(t, "textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		}),
	)

	diagnostics := diagnosticsFromLastNotification(t, messages)
	if len(diagnostics) != 0 {
		t.Fatalf("close diagnostics = %d, want empty clear", len(diagnostics))
	}
}

func TestHoverAndCompletionUseAnalyzer(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "composition.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n    "

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": map[string]any{}}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtSubstring(t, text, "CompositeBucket", source.EncodingUTF16),
		}),
		requestFrame(t, 3, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	hover := resultMap(t, responseForID(t, messages, 2))
	contents := asMap(t, hover["contents"])
	if !strings.Contains(contents["value"].(string), "Composite kind") {
		t.Fatalf("hover contents = %q, want analyzer docs", contents["value"])
	}
	completion := resultMap(t, responseForID(t, messages, 3))
	if !itemsContainLabel(asSlice(t, completion["items"]), "kind") {
		t.Fatalf("completion items = %#v, want kind", completion["items"])
	}
}

func TestCompletionItemsIncludePresentationMetadata(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "completion-presentation.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n\n"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": map[string]any{}}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	items := asSlice(t, completion["items"])
	if got, want := completionLabelsForTest(t, items), []string{"apiVersion", "kind", "metadata", "spec"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("completion labels = %#v, want %#v", got, want)
	}
	for _, raw := range items {
		item := asMap(t, raw)
		if item["kind"] != float64(10) {
			t.Fatalf("completion item %#v kind = %#v, want LSP Property 10", item["label"], item["kind"])
		}
		if item["detail"] != "Crossplane YAML field" {
			t.Fatalf("completion item %#v detail = %#v, want Crossplane YAML field", item["label"], item["detail"])
		}
	}
	item := completionItemByLabelForTest(t, items, "apiVersion")
	if item["documentation"] != "API version of the Composition resource." {
		t.Fatalf("apiVersion documentation = %#v, want existing analyzer documentation", item["documentation"])
	}
	if item["detail"] == item["documentation"] {
		t.Fatalf("detail was copied from documentation: %#v", item)
	}
}

func TestHoverAndCompletionUseNegotiatedUTF8Positions(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "utf8.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    k\nmetadata:\n  name: demo\n  emoji: \"😀\"\n"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{
			"rootUri": fileURI(root),
			"capabilities": map[string]any{
				"general": map[string]any{"positionEncodings": []string{"utf-8", "utf-16"}},
			},
		}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtSubstring(t, text, "name: demo", source.EncodingUTF8),
		}),
		requestFrame(t, 3, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, strings.Index(text, "    k")+len("    k"), source.EncodingUTF8),
		}),
	)

	hover := resultMap(t, responseForID(t, messages, 2))
	contents := asMap(t, hover["contents"])
	if !strings.Contains(contents["value"].(string), "Name of the Composition") {
		t.Fatalf("UTF-8 hover contents = %q", contents["value"])
	}
	completion := resultMap(t, responseForID(t, messages, 3))
	if !itemsContainLabel(asSlice(t, completion["items"]), "kind") {
		t.Fatalf("UTF-8 completion items = %#v, want kind", completion["items"])
	}
}

func TestCompletionItemsIncludePlainTextEdits(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "completion-edit.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nmetadata:\n  name: root-composition\ns"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": map[string]any{}}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	item := completionItemByLabelForTest(t, asSlice(t, completion["items"]), "spec")
	edit := asMap(t, item["textEdit"])
	if edit["newText"] != "spec:" {
		t.Fatalf("newText = %#v, want spec:", edit["newText"])
	}
	if _, ok := item["insertTextFormat"]; ok {
		t.Fatalf("completion should not use snippets: %#v", item)
	}
	if _, ok := item["insertTextMode"]; ok {
		t.Fatalf("completion should not advertise insertTextMode without client support: %#v", item)
	}
	rng := asMap(t, edit["range"])
	start := asMap(t, rng["start"])
	end := asMap(t, rng["end"])
	if start["line"] != float64(4) || start["character"] != float64(0) || end["line"] != float64(4) || end["character"] != float64(1) {
		t.Fatalf("textEdit range = %#v, want line 4 char 0..1", rng)
	}
}

func TestCompletionTextEditCorrectsIndentedRootKey(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "completion-package-root-edit.yaml"))
	text := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: root-package\n  s"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{
			"rootUri":      fileURI(root),
			"capabilities": zedCompletionCapabilities(),
		}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	item := completionItemByLabelForTest(t, asSlice(t, completion["items"]), "spec")
	edit := asMap(t, item["textEdit"])
	if edit["newText"] != "spec:" {
		t.Fatalf("newText = %#v, want spec:", edit["newText"])
	}
	if item["insertTextMode"] != float64(1) {
		t.Fatalf("insertTextMode = %#v, want asIs", item["insertTextMode"])
	}
	rng := asMap(t, edit["range"])
	start := asMap(t, rng["start"])
	end := asMap(t, rng["end"])
	if start["line"] != float64(4) || start["character"] != float64(0) || end["line"] != float64(4) || end["character"] != float64(3) {
		t.Fatalf("textEdit range = %#v, want line 4 char 0..3", rng)
	}
}

func TestCompletionTextEditPreservesZeroWidthInsertionRange(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "completion-blank-line-edit.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n\nmetadata:\n  name: root-composition\n"
	blankLineOffset := strings.Index(text, "\n\n")
	if blankLineOffset < 0 {
		t.Fatal("test setup: blank line not found")
	}
	blankLineOffset++

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": map[string]any{}}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, blankLineOffset, source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	item := completionItemByLabelForTest(t, asSlice(t, completion["items"]), "spec")
	edit := asMap(t, item["textEdit"])
	if edit["newText"] != "spec:" {
		t.Fatalf("newText = %#v, want spec:", edit["newText"])
	}
	rng := asMap(t, edit["range"])
	start := asMap(t, rng["start"])
	end := asMap(t, rng["end"])
	if start["line"] != float64(2) || start["character"] != float64(0) || end["line"] != float64(2) || end["character"] != float64(0) {
		t.Fatalf("zero-width textEdit range = %#v, want line 2 char 0..0", rng)
	}
}

func TestDiagnosticsMapAnalyzerSpansToLaterLineRanges(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "malformed.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\nbroken: [unterminated\n"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{
			"rootUri": fileURI(root),
			"capabilities": map[string]any{
				"general": map[string]any{"positionEncodings": []string{"utf-8"}},
			},
		}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
	)

	diagnostics := diagnosticsFromLastNotification(t, messages)
	if len(diagnostics) == 0 {
		t.Fatalf("expected malformed YAML diagnostics, messages=%#v", messages)
	}
	diagnostic := asMap(t, diagnostics[0])
	rng := asMap(t, diagnostic["range"])
	start := asMap(t, rng["start"])
	end := asMap(t, rng["end"])
	if start["line"] == float64(0) && start["character"] == float64(0) &&
		end["line"] == float64(0) && end["character"] == float64(1) {
		t.Fatalf("diagnostic collapsed to start of file: %#v", diagnostic["range"])
	}
	if start["line"].(float64) < 4 {
		t.Fatalf("diagnostic start line = %v, want later malformed line; range=%#v", start["line"], diagnostic["range"])
	}
}

func TestZeroWidthDiagnosticRangeUsesDiagnosticOffset(t *testing.T) {
	s := NewServer(bytes.NewReader(nil), io.Discard, io.Discard)
	text := "first\nsecond\n"
	offset := strings.Index(text, "cond")

	rng := s.rangeFromSpan(text, analyzer.Span{Start: offset, End: offset})

	if rng.Start.Line != 1 || rng.Start.Character != 2 {
		t.Fatalf("range start = %#v, want line 1 character 2", rng.Start)
	}
	if rng.End == rng.Start {
		t.Fatalf("zero-width diagnostic range was not expanded: %#v", rng)
	}
}

func TestStaleDiagnosticPublicationIsDropped(t *testing.T) {
	var out bytes.Buffer
	s, uri := newServerWithAnalyzer(t, &out)
	doc := s.analyzer.OpenDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nbroken: [unterminated\n")
	publication, ok := s.diagnosticsPublication(uri, doc.Generation)
	if !ok || len(publication.Diagnostics) == 0 {
		t.Fatalf("expected diagnostic publication, ok=%v diagnostics=%d", ok, len(publication.Diagnostics))
	}

	s.analyzer.ChangeDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n")
	if err := s.publishDiagnostics(publication); err != nil {
		t.Fatalf("publish stale diagnostics: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("stale diagnostics were published: %s", out.String())
	}

	doc = s.analyzer.OpenDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nbroken: [unterminated\n")
	publication, ok = s.diagnosticsPublication(uri, doc.Generation)
	if !ok {
		t.Fatal("expected diagnostic publication before close")
	}
	s.analyzer.CloseDocument(uri)
	if err := s.publishDiagnostics(publication); err != nil {
		t.Fatalf("publish diagnostics for closed document: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("closed document diagnostics were published: %s", out.String())
	}
}

func TestStaleHoverAndCompletionSnapshotsAreDropped(t *testing.T) {
	var out bytes.Buffer
	s, uri := newServerWithAnalyzer(t, &out)
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n"
	s.analyzer.OpenDocument(uri, text)

	hoverSnapshot, ok := s.positionSnapshot(uri, toLSPPosition(source.PositionAtByteOffset(text, strings.Index(text, "CompositeBucket"), source.EncodingUTF16)))
	if !ok {
		t.Fatal("expected hover position snapshot")
	}
	completionSnapshot, ok := s.positionSnapshot(uri, toLSPPosition(source.PositionAtByteOffset(text, strings.Index(text, "compositeTypeRef"), source.EncodingUTF16)))
	if !ok {
		t.Fatal("expected completion position snapshot")
	}

	s.analyzer.ChangeDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n")
	if hover, ok := s.hoverForSnapshot(hoverSnapshot); ok {
		t.Fatalf("stale hover returned %#v", hover)
	}
	if completion, ok := s.completionForSnapshot(completionSnapshot); ok {
		t.Fatalf("stale completion returned %#v", completion)
	}
}

func runServerFrames(t *testing.T, frames ...string) []Message {
	t.Helper()
	var in bytes.Buffer
	for _, frame := range frames {
		in.WriteString(frame)
	}
	in.WriteString(notificationFrame(t, "exit", nil))

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := NewServer(&in, &out, &errOut).Run()
	if code != 0 {
		t.Fatalf("server exit code = %d stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("server wrote stderr: %s", errOut.String())
	}
	return readMessages(t, out.Bytes())
}

func requestFrame(t *testing.T, id int, method string, params any) string {
	t.Helper()
	payload := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	return jsonFrame(t, payload)
}

func notificationFrame(t *testing.T, method string, params any) string {
	t.Helper()
	payload := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		payload["params"] = params
	}
	return jsonFrame(t, payload)
}

func jsonFrame(t *testing.T, payload any) string {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	return "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + string(body)
}

func readMessages(t *testing.T, data []byte) []Message {
	t.Helper()
	reader := bufio.NewReader(bytes.NewReader(data))
	var messages []Message
	for {
		msg, err := ReadMessage(reader)
		if err == io.EOF {
			return messages
		}
		if err != nil {
			t.Fatalf("read output message: %v\noutput:\n%s", err, string(data))
		}
		messages = append(messages, msg)
	}
}

func responseForID(t *testing.T, messages []Message, id int) Message {
	t.Helper()
	for _, message := range messages {
		if sameID(message.ID, id) {
			return message
		}
	}
	t.Fatalf("missing response id %d in %#v", id, messages)
	return Message{}
}

func diagnosticsFromLastNotification(t *testing.T, messages []Message) []any {
	t.Helper()
	var diagnostics []any
	found := false
	for _, message := range messages {
		if message.Method != "textDocument/publishDiagnostics" {
			continue
		}
		params := paramsMap(t, message)
		diagnostics = asSlice(t, params["diagnostics"])
		found = true
	}
	if !found {
		t.Fatalf("missing diagnostics notification in %#v", messages)
	}
	return diagnostics
}

func resultMap(t *testing.T, msg Message) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		t.Fatalf("unmarshal result %q: %v", string(msg.Result), err)
	}
	return result
}

func paramsMap(t *testing.T, msg Message) map[string]any {
	t.Helper()
	var params map[string]any
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Fatalf("unmarshal params %q: %v", string(msg.Params), err)
	}
	return params
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value %#v is %T, want object", value, value)
	}
	return result
}

func asSlice(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("value %#v is %T, want array", value, value)
	}
	return result
}

func containsString(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func itemsContainLabel(items []any, label string) bool {
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if ok && item["label"] == label {
			return true
		}
	}
	return false
}

func completionLabelsForTest(t *testing.T, items []any) []string {
	t.Helper()
	labels := make([]string, 0, len(items))
	for _, raw := range items {
		item := asMap(t, raw)
		label, ok := item["label"].(string)
		if !ok {
			t.Fatalf("completion item label = %#v, want string", item["label"])
		}
		labels = append(labels, label)
	}
	return labels
}

func completionItemByLabelForTest(t *testing.T, items []any, label string) map[string]any {
	t.Helper()
	for _, raw := range items {
		item := asMap(t, raw)
		if item["label"] == label {
			return item
		}
	}
	t.Fatalf("completion item %q not found in %#v", label, items)
	return nil
}

func zedCompletionCapabilities() map[string]any {
	return map[string]any{
		"textDocument": map[string]any{
			"completion": map[string]any{
				"completionItem": map[string]any{
					"insertTextModeSupport": map[string]any{
						"valueSet": []int{1, 2},
					},
				},
				"insertTextMode": 2,
			},
		},
	}
}

func sameID(got any, want int) bool {
	switch value := got.(type) {
	case float64:
		return value == float64(want)
	case int:
		return value == want
	case string:
		return value == strconv.Itoa(want)
	default:
		return false
	}
}

func positionAtSubstring(t *testing.T, text string, needle string, encoding source.Encoding) map[string]any {
	t.Helper()
	offset := strings.Index(text, needle)
	if offset < 0 {
		t.Fatalf("substring %q not found", needle)
	}
	position := source.PositionAtByteOffset(text, offset, encoding)
	return map[string]any{"line": position.Line, "character": position.Character}
}

func positionAtOffset(t *testing.T, text string, offset int, encoding source.Encoding) map[string]any {
	t.Helper()
	position := source.PositionAtByteOffset(text, offset, encoding)
	return map[string]any{"line": position.Line, "character": position.Character}
}

func toLSPPosition(position source.Position) Position {
	return Position{Line: position.Line, Character: position.Character}
}

func fileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func testRoot(t *testing.T) string {
	t.Helper()
	return testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
}

func newServerWithAnalyzer(t *testing.T, out *bytes.Buffer) (*Server, string) {
	t.Helper()
	root := testRoot(t)
	a, err := analyzer.New(analyzer.Options{WorkspaceRoot: root, Limits: analyzer.DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	s := NewServer(bytes.NewReader(nil), out, io.Discard)
	s.analyzer = a
	return s, fileURI(filepath.Join(root, "api", "composition.yaml"))
}
