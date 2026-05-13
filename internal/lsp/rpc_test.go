package lsp

import (
	"bufio"
	"bytes"
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

func TestReadMessageRequiresContentLength(t *testing.T) {
	_, err := ReadMessage(bufio.NewReader(bytes.NewBufferString("\r\n{}")))
	if err == nil {
		t.Fatal("expected missing Content-Length error")
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
