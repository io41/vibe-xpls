package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestReadWriteMessage(t *testing.T) {
	var out bytes.Buffer
	err := WriteMessage(&out, Message{JSONRPC: "2.0", ID: 1, Method: "initialize"})
	if err != nil {
		t.Fatalf("write message: %v", err)
	}

	msg, err := ReadMessage(bufio.NewReader(&out))
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if msg.Method != "initialize" || msg.ID.(float64) != 1 {
		t.Fatalf("message = %#v", msg)
	}
}

func TestWriteMessageEncodesNullResult(t *testing.T) {
	var out bytes.Buffer
	err := WriteMessage(&out, Message{ID: 1, Result: json.RawMessage("null")})
	if err != nil {
		t.Fatalf("write message: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"result":null`)) {
		t.Fatalf("message does not include null result: %s", out.String())
	}

	msg, err := ReadMessage(bufio.NewReader(&out))
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if string(msg.Result) != "null" {
		t.Fatalf("result = %q, want null", string(msg.Result))
	}
}

func TestReadMessageReadsMultipleMessages(t *testing.T) {
	var out bytes.Buffer
	if err := WriteMessage(&out, Message{Method: "initialize"}); err != nil {
		t.Fatalf("write initialize: %v", err)
	}
	if err := WriteMessage(&out, Message{Method: "shutdown"}); err != nil {
		t.Fatalf("write shutdown: %v", err)
	}

	reader := bufio.NewReader(&out)
	first, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("read first message: %v", err)
	}
	second, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("read second message: %v", err)
	}
	if first.Method != "initialize" || second.Method != "shutdown" {
		t.Fatalf("methods = %q, %q", first.Method, second.Method)
	}
}

func TestReadMessageRequiresContentLength(t *testing.T) {
	_, err := ReadMessage(bufio.NewReader(bytes.NewBufferString("\r\n{}")))
	if err == nil {
		t.Fatal("expected missing Content-Length error")
	}
}

func TestReadMessageRejectsOversizedContentLength(t *testing.T) {
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n{}", maxMessageSize+1)
	_, err := ReadMessage(bufio.NewReader(bytes.NewBufferString(input)))
	if err == nil {
		t.Fatal("expected oversized Content-Length error")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error = %q, want exceeds", err)
	}
}

func TestReadMessageAcceptsLowercaseContentLength(t *testing.T) {
	msg, err := ReadMessage(bufio.NewReader(bytes.NewBufferString("content-length: 37\r\n\r\n{\"jsonrpc\":\"2.0\",\"method\":\"shutdown\"}")))
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if msg.Method != "shutdown" {
		t.Fatalf("method = %q, want shutdown", msg.Method)
	}
}

func TestReadMessageRejectsMalformedHeader(t *testing.T) {
	_, err := ReadMessage(bufio.NewReader(bytes.NewBufferString("Content-Length 2\r\n\r\n{}")))
	if err == nil {
		t.Fatal("expected malformed header error")
	}
}

func TestReadMessageRejectsMalformedContentLength(t *testing.T) {
	_, err := ReadMessage(bufio.NewReader(bytes.NewBufferString("Content-Length: nope\r\n\r\n{}")))
	if err == nil {
		t.Fatal("expected malformed Content-Length error")
	}
}
