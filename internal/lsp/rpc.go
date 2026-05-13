package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxMessageSize = 8 * 1024 * 1024

func ReadMessage(r *bufio.Reader) (Message, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return Message{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return Message{}, fmt.Errorf("malformed header %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return Message{}, err
			}
			if parsed < 0 {
				return Message{}, fmt.Errorf("invalid Content-Length %d", parsed)
			}
			if parsed > maxMessageSize {
				return Message{}, fmt.Errorf("Content-Length %d exceeds maximum %d", parsed, maxMessageSize)
			}
			length = parsed
		}
	}
	if length < 0 {
		return Message{}, errors.New("missing Content-Length header")
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return Message{}, err
	}
	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

func WriteMessage(w io.Writer, msg Message) error {
	if msg.JSONRPC == "" {
		msg.JSONRPC = "2.0"
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}
