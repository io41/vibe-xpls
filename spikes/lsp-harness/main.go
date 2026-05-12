package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type server struct {
	in   *bufio.Reader
	out  io.Writer
	docs map[string]string
	mu   sync.Mutex
}

func main() {
	s := newServer(os.Stdin, os.Stdout)
	if err := s.run(); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newServer(in io.Reader, out io.Writer) *server {
	return &server{
		in:   bufio.NewReader(in),
		out:  out,
		docs: map[string]string{},
	}
}

func (s *server) run() error {
	for {
		msg, err := readMessage(s.in)
		if err != nil {
			return err
		}

		if msg.Method == "exit" {
			return nil
		}

		if msg.Method == "" {
			continue
		}

		if err := s.handle(msg); err != nil {
			return err
		}
	}
}

func (s *server) handle(msg message) error {
	switch msg.Method {
	case "initialize":
		return s.respond(msg.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync": 1,
				"hoverProvider":    true,
				"completionProvider": map[string]any{
					"triggerCharacters": []string{".", ":"},
				},
			},
			"serverInfo": map[string]any{
				"name":    "vibe-xpls-lsp-harness",
				"version": "0.0.1-spike",
			},
		})
	case "shutdown":
		return s.respond(msg.ID, nil)
	case "textDocument/didOpen":
		uri, text, err := didOpenParams(msg.Params)
		if err != nil {
			return s.requestError(msg.ID, -32602, err.Error())
		}
		s.setDocument(uri, text)
		return s.publishDiagnostics(uri, text)
	case "textDocument/didChange":
		uri, text, err := didChangeParams(msg.Params)
		if err != nil {
			return s.requestError(msg.ID, -32602, err.Error())
		}
		s.setDocument(uri, text)
		return s.publishDiagnostics(uri, text)
	case "textDocument/didClose":
		uri, err := documentURIParam(msg.Params)
		if err != nil {
			return s.requestError(msg.ID, -32602, err.Error())
		}
		s.deleteDocument(uri)
		return s.publishDiagnostics(uri, "")
	case "textDocument/hover":
		return s.respond(msg.ID, map[string]any{
			"contents": map[string]any{
				"kind":  "markdown",
				"value": "vibe-xpls LSP harness hover",
			},
		})
	case "textDocument/completion":
		return s.respond(msg.ID, map[string]any{
			"isIncomplete": false,
			"items": []map[string]any{
				{"label": "apiVersion", "kind": 10},
				{"label": "kind", "kind": 10},
				{"label": "metadata", "kind": 10},
			},
		})
	default:
		if msg.ID != nil {
			return s.requestError(msg.ID, -32601, "method not found")
		}
		return nil
	}
}

func (s *server) setDocument(uri, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = text
}

func (s *server) deleteDocument(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

func (s *server) publishDiagnostics(uri, text string) error {
	diagnostics := []map[string]any{}
	if strings.Contains(text, "xpls-spike-error") {
		diagnostics = append(diagnostics, map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 15},
			},
			"severity": 1,
			"source":   "vibe-xpls-lsp-harness",
			"message":  "deterministic spike diagnostic",
		})
	}

	return s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri":         uri,
		"diagnostics": diagnostics,
	})
}

func (s *server) respond(id any, result any) error {
	return writeMessage(s.out, message{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *server) requestError(id any, code int, text string) error {
	return writeMessage(s.out, message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &responseError{
			Code:    code,
			Message: text,
		},
	})
}

func (s *server) notify(method string, params any) error {
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return writeMessage(s.out, message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  payload,
	})
}

func readMessage(r *bufio.Reader) (message, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return message{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return message{}, fmt.Errorf("malformed header %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return message{}, err
			}
			length = parsed
		}
	}

	if length < 0 {
		return message{}, errors.New("missing Content-Length header")
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return message{}, err
	}

	var msg message
	if err := json.Unmarshal(body, &msg); err != nil {
		return message{}, err
	}
	return msg, nil
}

func writeMessage(w io.Writer, msg message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(body), body)
	return err
}

func didOpenParams(raw json.RawMessage) (string, string, error) {
	var params struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return "", "", err
	}
	if params.TextDocument.URI == "" {
		return "", "", errors.New("missing textDocument.uri")
	}
	return params.TextDocument.URI, params.TextDocument.Text, nil
}

func didChangeParams(raw json.RawMessage) (string, string, error) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return "", "", err
	}
	if params.TextDocument.URI == "" {
		return "", "", errors.New("missing textDocument.uri")
	}
	if len(params.ContentChanges) == 0 {
		return "", "", errors.New("missing contentChanges")
	}
	return params.TextDocument.URI, params.ContentChanges[len(params.ContentChanges)-1].Text, nil
}

func documentURIParam(raw json.RawMessage) (string, error) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return "", err
	}
	if params.TextDocument.URI == "" {
		return "", errors.New("missing textDocument.uri")
	}
	return params.TextDocument.URI, nil
}
