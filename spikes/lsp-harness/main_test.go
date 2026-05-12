package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestInitializeReturnsCapabilities(t *testing.T) {
	messages := runHarness(t, encodeRequest(1, "initialize", map[string]any{}))

	response := findResponse(t, messages, 1)
	result := asMap(t, response.Result)
	capabilities := asMap(t, result["capabilities"])

	if capabilities["hoverProvider"] != true {
		t.Fatalf("expected hoverProvider true, got %#v", capabilities["hoverProvider"])
	}
	if _, ok := capabilities["completionProvider"].(map[string]any); !ok {
		t.Fatalf("expected completionProvider object, got %#v", capabilities["completionProvider"])
	}
}

func TestDidOpenPublishesDeterministicDiagnostic(t *testing.T) {
	messages := runHarness(t,
		encodeNotification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":  "file:///composition.yaml",
				"text": "xpls-spike-error",
			},
		}),
	)

	notification := findNotification(t, messages, "textDocument/publishDiagnostics")
	params := rawAsMap(t, notification.Params)
	diagnostics := params["diagnostics"].([]any)

	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %d", len(diagnostics))
	}
	diagnostic := diagnostics[0].(map[string]any)
	if diagnostic["message"] != "deterministic spike diagnostic" {
		t.Fatalf("unexpected diagnostic message: %#v", diagnostic["message"])
	}
}

func TestHoverReturnsContent(t *testing.T) {
	messages := runHarness(t,
		encodeRequest(2, "textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": "file:///composition.yaml"},
			"position":     map[string]any{"line": 0, "character": 0},
		}),
	)

	response := findResponse(t, messages, 2)
	result := asMap(t, response.Result)
	contents := asMap(t, result["contents"])

	if contents["value"] == "" {
		t.Fatalf("expected hover contents, got %#v", contents)
	}
}

func TestCompletionReturnsItems(t *testing.T) {
	messages := runHarness(t,
		encodeRequest(3, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": "file:///composition.yaml"},
			"position":     map[string]any{"line": 0, "character": 0},
		}),
	)

	response := findResponse(t, messages, 3)
	result := asMap(t, response.Result)
	items := result["items"].([]any)

	if len(items) == 0 {
		t.Fatal("expected at least one completion item")
	}
}

func TestShutdownReturnsResponse(t *testing.T) {
	messages := runHarness(t, encodeRequest(4, "shutdown", nil))

	response := findResponse(t, messages, 4)
	if response.Error != nil {
		t.Fatalf("expected shutdown success, got error %#v", response.Error)
	}
}

func TestDidChangeAndDidClosePublishDiagnostics(t *testing.T) {
	messages := runHarness(t,
		encodeNotification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":  "file:///composition.yaml",
				"text": "ok",
			},
		}),
		encodeNotification("textDocument/didChange", map[string]any{
			"textDocument": map[string]any{"uri": "file:///composition.yaml"},
			"contentChanges": []map[string]any{
				{"text": "xpls-spike-error"},
			},
		}),
		encodeNotification("textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": "file:///composition.yaml"},
		}),
	)

	diagnostics := findNotifications(t, messages, "textDocument/publishDiagnostics")
	if len(diagnostics) != 3 {
		t.Fatalf("expected three diagnostic notifications, got %d", len(diagnostics))
	}

	changed := rawAsMap(t, diagnostics[1].Params)
	if got := len(changed["diagnostics"].([]any)); got != 1 {
		t.Fatalf("expected changed document diagnostic, got %d", got)
	}

	closed := rawAsMap(t, diagnostics[2].Params)
	if got := len(closed["diagnostics"].([]any)); got != 0 {
		t.Fatalf("expected close to clear diagnostics, got %d", got)
	}
}

func runHarness(t *testing.T, frames ...[]byte) []message {
	t.Helper()

	var in bytes.Buffer
	for _, frame := range frames {
		in.Write(frame)
	}
	in.Write(encodeNotification("exit", nil))

	var out bytes.Buffer
	s := newServer(&in, &out)
	if err := s.run(); err != nil {
		t.Fatalf("server failed: %v", err)
	}

	messages, err := decodeMessages(out.Bytes())
	if err != nil {
		t.Fatalf("decode output messages: %v", err)
	}
	return messages
}

func findResponse(t *testing.T, messages []message, id int) message {
	t.Helper()

	for _, msg := range messages {
		if msg.ID == float64(id) {
			return msg
		}
	}
	t.Fatalf("response id %d not found in %#v", id, messages)
	return message{}
}

func findNotification(t *testing.T, messages []message, method string) message {
	t.Helper()

	matches := findNotifications(t, messages, method)
	if len(matches) == 0 {
		t.Fatalf("notification %q not found in %#v", method, messages)
	}
	return matches[0]
}

func findNotifications(t *testing.T, messages []message, method string) []message {
	t.Helper()

	var matches []message
	for _, msg := range messages {
		if msg.Method == method {
			matches = append(matches, msg)
		}
	}
	return matches
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()

	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %#v", value)
	}
	return m
}

func rawAsMap(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal raw map: %v", err)
	}
	return value
}

func encodeRequest(id int, method string, params any) []byte {
	body, err := json.Marshal(message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  mustRaw(params),
	})
	if err != nil {
		panic(err)
	}
	return frame(body)
}

func encodeNotification(method string, params any) []byte {
	body, err := json.Marshal(message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  mustRaw(params),
	})
	if err != nil {
		panic(err)
	}
	return frame(body)
}

func frame(body []byte) []byte {
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body))
}

func mustRaw(params any) json.RawMessage {
	if params == nil {
		return nil
	}
	body, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	return body
}

func decodeMessages(out []byte) ([]message, error) {
	reader := bufio.NewReader(bytes.NewReader(out))
	var messages []message
	for {
		msg, err := readMessage(reader)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}
