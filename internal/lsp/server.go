package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	"github.com/io41/vibe-xpls/internal/analyzer"
	"github.com/io41/vibe-xpls/internal/app"
	"github.com/io41/vibe-xpls/internal/source"
)

const (
	methodNotFound = -32601
	invalidParams  = -32602
)

var nullResult = json.RawMessage("null")

type Server struct {
	in               *bufio.Reader
	out              io.Writer
	errOut           io.Writer
	analyzer         *analyzer.Analyzer
	positionEncoding source.Encoding
}

type initializeParams struct {
	RootURI      string `json:"rootUri"`
	RootPath     string `json:"rootPath"`
	Capabilities struct {
		General struct {
			PositionEncodings []string `json:"positionEncodings"`
		} `json:"general"`
	} `json:"capabilities"`
}

type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
	ServerInfo   serverInfo         `json:"serverInfo"`
}

type serverCapabilities struct {
	TextDocumentSync   int               `json:"textDocumentSync"`
	HoverProvider      bool              `json:"hoverProvider"`
	CompletionProvider completionOptions `json:"completionProvider"`
	PositionEncoding   string            `json:"positionEncoding"`
}

type completionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type textDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type didChangeParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type textDocumentParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

type positionedParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position Position `json:"position"`
}

type diagnosticPublication struct {
	URI         string
	Generation  analyzer.Generation
	Text        string
	Diagnostics []analyzer.Diagnostic
}

type documentPositionSnapshot struct {
	URI        string
	Generation analyzer.Generation
	Text       string
	Position   Position
	Offset     int
}

type hoverResponse struct {
	Contents markupContent `json:"contents"`
}

type markupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type completionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []completionItem `json:"items"`
}

type completionItem struct {
	Label         string    `json:"label"`
	Documentation string    `json:"documentation,omitempty"`
	TextEdit      *textEdit `json:"textEdit,omitempty"`
}

type textEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

func NewServer(in io.Reader, out io.Writer, errOut io.Writer) *Server {
	return &Server{
		in:               bufio.NewReader(in),
		out:              out,
		errOut:           errOut,
		positionEncoding: source.EncodingUTF16,
	}
}

func (s *Server) Run() int {
	for {
		msg, err := ReadMessage(s.in)
		if errors.Is(err, io.EOF) {
			return 0
		}
		if err != nil {
			fmt.Fprintln(s.errOut, err)
			return 1
		}
		if msg.Method == "exit" {
			return 0
		}
		if err := s.handle(msg); err != nil {
			fmt.Fprintln(s.errOut, err)
			return 1
		}
	}
}

func (s *Server) handle(msg Message) error {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "shutdown":
		return s.respond(msg.ID, nil, nil)
	case "textDocument/didOpen":
		return s.handleDidOpen(msg.Params)
	case "textDocument/didChange":
		return s.handleDidChange(msg.Params)
	case "textDocument/didClose":
		return s.handleDidClose(msg.Params)
	case "textDocument/hover":
		return s.handleHover(msg)
	case "textDocument/completion":
		return s.handleCompletion(msg)
	default:
		if msg.ID != nil {
			return s.respond(msg.ID, nil, &ResponseError{Code: methodNotFound, Message: "method not found"})
		}
		return nil
	}
}

func (s *Server) handleInitialize(msg Message) error {
	var params initializeParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return s.respond(msg.ID, nil, &ResponseError{Code: invalidParams, Message: err.Error()})
	}
	s.positionEncoding = negotiatePositionEncoding(params.Capabilities.General.PositionEncodings)

	a, err := analyzer.New(analyzer.Options{WorkspaceRoot: rootFromInitialize(params), Limits: analyzer.DefaultLimits()})
	if err != nil {
		return s.respond(msg.ID, nil, &ResponseError{Code: invalidParams, Message: err.Error()})
	}
	s.analyzer = a

	return s.respond(msg.ID, initializeResult{
		Capabilities: serverCapabilities{
			TextDocumentSync:   1,
			HoverProvider:      true,
			CompletionProvider: completionOptions{TriggerCharacters: []string{"\n"}},
			PositionEncoding:   string(s.positionEncoding),
		},
		ServerInfo: serverInfo{Name: "vibe-xpls", Version: app.Version()},
	}, nil)
}

func (s *Server) handleDidOpen(raw json.RawMessage) error {
	if s.analyzer == nil {
		return nil
	}
	var params didOpenParams
	if err := decodeParams(raw, &params); err != nil || params.TextDocument.URI == "" {
		return nil
	}
	doc := s.analyzer.OpenDocument(params.TextDocument.URI, params.TextDocument.Text)
	return s.publishDiagnosticsForGeneration(doc.URI, doc.Generation)
}

func (s *Server) handleDidChange(raw json.RawMessage) error {
	if s.analyzer == nil {
		return nil
	}
	var params didChangeParams
	if err := decodeParams(raw, &params); err != nil || params.TextDocument.URI == "" || len(params.ContentChanges) == 0 {
		return nil
	}
	text := params.ContentChanges[len(params.ContentChanges)-1].Text
	doc := s.analyzer.ChangeDocument(params.TextDocument.URI, text)
	return s.publishDiagnosticsForGeneration(doc.URI, doc.Generation)
}

func (s *Server) handleDidClose(raw json.RawMessage) error {
	var params textDocumentParams
	if err := decodeParams(raw, &params); err != nil || params.TextDocument.URI == "" {
		return nil
	}
	if s.analyzer != nil {
		s.analyzer.CloseDocument(params.TextDocument.URI)
	}
	return s.publishEmptyDiagnostics(params.TextDocument.URI)
}

func (s *Server) handleHover(msg Message) error {
	var params positionedParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return s.respond(msg.ID, nil, &ResponseError{Code: invalidParams, Message: err.Error()})
	}
	snapshot, ok := s.positionSnapshot(params.TextDocument.URI, params.Position)
	if !ok {
		return s.respond(msg.ID, nil, nil)
	}
	hover, ok := s.hoverForSnapshot(snapshot)
	if !ok {
		return s.respond(msg.ID, nil, nil)
	}
	return s.respond(msg.ID, hoverResponse{
		Contents: markupContent{Kind: "markdown", Value: hover.Markdown},
	}, nil)
}

func (s *Server) handleCompletion(msg Message) error {
	var params positionedParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return s.respond(msg.ID, nil, &ResponseError{Code: invalidParams, Message: err.Error()})
	}
	snapshot, ok := s.positionSnapshot(params.TextDocument.URI, params.Position)
	if !ok {
		return s.respond(msg.ID, emptyCompletionList(), nil)
	}
	completion, ok := s.completionForSnapshot(snapshot)
	if !ok {
		return s.respond(msg.ID, emptyCompletionList(), nil)
	}
	items := make([]completionItem, 0, len(completion.Items))
	for _, item := range completion.Items {
		out := completionItem{Label: item.Label, Documentation: item.Documentation}
		if item.TextEdit != nil {
			out.TextEdit = &textEdit{
				Range:   s.rangeFromTextEditSpan(snapshot.Text, item.TextEdit.Replace),
				NewText: item.TextEdit.NewText,
			}
		}
		items = append(items, out)
	}
	return s.respond(msg.ID, completionList{IsIncomplete: false, Items: items}, nil)
}

func (s *Server) publishDiagnosticsForGeneration(uri string, generation analyzer.Generation) error {
	publication, ok := s.diagnosticsPublication(uri, generation)
	if !ok {
		return nil
	}
	return s.publishDiagnostics(publication)
}

func (s *Server) diagnosticsPublication(uri string, generation analyzer.Generation) (diagnosticPublication, bool) {
	if s.analyzer == nil {
		return diagnosticPublication{}, false
	}
	doc, ok := s.analyzer.Document(uri)
	if !ok || doc.Generation != generation {
		return diagnosticPublication{}, false
	}
	return diagnosticPublication{
		URI:         uri,
		Generation:  generation,
		Text:        doc.Text,
		Diagnostics: s.analyzer.Diagnostics(uri),
	}, true
}

func (s *Server) publishDiagnostics(publication diagnosticPublication) error {
	if !s.documentGenerationMatches(publication.URI, publication.Generation) {
		return nil
	}
	items := make([]diagnostic, 0, len(publication.Diagnostics))
	for _, item := range publication.Diagnostics {
		items = append(items, diagnostic{
			Range:    s.rangeFromSpan(publication.Text, item.Span),
			Severity: diagnosticSeverity(item.Severity),
			Source:   item.Source,
			Message:  item.Message,
		})
	}
	return s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri":         publication.URI,
		"diagnostics": items,
	})
}

func (s *Server) publishEmptyDiagnostics(uri string) error {
	return s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri":         uri,
		"diagnostics": []diagnostic{},
	})
}

func (s *Server) positionSnapshot(uri string, position Position) (documentPositionSnapshot, bool) {
	if s.analyzer == nil || uri == "" {
		return documentPositionSnapshot{}, false
	}
	doc, ok := s.analyzer.Document(uri)
	if !ok {
		return documentPositionSnapshot{}, false
	}
	offset := source.ByteOffsetAtPosition(doc.Text, source.Position{
		Line:      position.Line,
		Character: position.Character,
	}, s.positionEncoding)
	return documentPositionSnapshot{
		URI:        uri,
		Generation: doc.Generation,
		Text:       doc.Text,
		Position:   position,
		Offset:     offset,
	}, true
}

func (s *Server) hoverForSnapshot(snapshot documentPositionSnapshot) (analyzer.Hover, bool) {
	if !s.documentGenerationMatches(snapshot.URI, snapshot.Generation) {
		return analyzer.Hover{}, false
	}
	hover, ok := s.analyzer.HoverAtOffset(snapshot.URI, snapshot.Offset)
	if !ok || !s.documentGenerationMatches(snapshot.URI, snapshot.Generation) {
		return analyzer.Hover{}, false
	}
	return hover, true
}

func (s *Server) completionForSnapshot(snapshot documentPositionSnapshot) (analyzer.Completion, bool) {
	if !s.documentGenerationMatches(snapshot.URI, snapshot.Generation) {
		return analyzer.Completion{}, false
	}
	completion := s.analyzer.CompletionAtOffset(snapshot.URI, snapshot.Offset)
	if !s.documentGenerationMatches(snapshot.URI, snapshot.Generation) {
		return analyzer.Completion{}, false
	}
	return completion, true
}

func (s *Server) documentGenerationMatches(uri string, generation analyzer.Generation) bool {
	if s.analyzer == nil {
		return false
	}
	doc, ok := s.analyzer.Document(uri)
	return ok && doc.Generation == generation
}

func (s *Server) rangeFromSpan(text string, span analyzer.Span) Range {
	start := clampOffset(span.Start, len(text))
	end := clampOffset(span.End, len(text))
	if end < start {
		end = start
	}

	startPosition := source.PositionAtByteOffset(text, start, s.positionEncoding)
	endPosition := source.PositionAtByteOffset(text, end, s.positionEncoding)
	if startPosition == endPosition {
		if end < len(text) {
			endPosition = source.PositionAtByteOffset(text, end+1, s.positionEncoding)
		}
		if startPosition == endPosition {
			endPosition.Character++
		}
	}
	return Range{
		Start: Position{Line: startPosition.Line, Character: startPosition.Character},
		End:   Position{Line: endPosition.Line, Character: endPosition.Character},
	}
}

func (s *Server) rangeFromTextEditSpan(text string, span analyzer.Span) Range {
	start := clampOffset(span.Start, len(text))
	end := clampOffset(span.End, len(text))
	if end < start {
		end = start
	}

	startPosition := source.PositionAtByteOffset(text, start, s.positionEncoding)
	endPosition := source.PositionAtByteOffset(text, end, s.positionEncoding)
	return Range{
		Start: Position{Line: startPosition.Line, Character: startPosition.Character},
		End:   Position{Line: endPosition.Line, Character: endPosition.Character},
	}
}

func (s *Server) respond(id any, result any, responseError *ResponseError) error {
	if id == nil {
		return nil
	}
	msg := Message{JSONRPC: "2.0", ID: id, Error: responseError}
	if responseError == nil {
		raw, err := marshalResult(result)
		if err != nil {
			return err
		}
		msg.Result = raw
	}
	return WriteMessage(s.out, msg)
}

func (s *Server) notify(method string, params any) error {
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return WriteMessage(s.out, Message{JSONRPC: "2.0", Method: method, Params: payload})
}

func marshalResult(result any) (json.RawMessage, error) {
	if result == nil {
		return nullResult, nil
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func decodeParams(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func negotiatePositionEncoding(offered []string) source.Encoding {
	for _, encoding := range offered {
		if encoding == string(source.EncodingUTF8) {
			return source.EncodingUTF8
		}
	}
	return source.EncodingUTF16
}

func rootFromInitialize(params initializeParams) string {
	if params.RootURI != "" {
		if path, ok := pathFromFileURI(params.RootURI); ok {
			return path
		}
	}
	if params.RootPath != "" {
		return filepath.Clean(params.RootPath)
	}
	return "."
}

func pathFromFileURI(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "file" || parsed.Path == "" {
		return "", false
	}
	if parsed.Host != "" && parsed.Host != "localhost" {
		return "", false
	}
	return filepath.Clean(filepath.FromSlash(parsed.Path)), true
}

func diagnosticSeverity(severity string) int {
	switch severity {
	case "warning":
		return 2
	case "info":
		return 3
	case "hint":
		return 4
	default:
		return 1
	}
}

func emptyCompletionList() completionList {
	return completionList{IsIncomplete: false, Items: []completionItem{}}
}

func clampOffset(offset int, length int) int {
	if offset < 0 {
		return 0
	}
	if offset > length {
		return length
	}
	return offset
}
