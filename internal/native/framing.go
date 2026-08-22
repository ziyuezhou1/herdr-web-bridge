// Package native implements Edge Native Messaging framing.
package native

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/protocol"
)

var (
	ErrMessageTooLarge = errors.New("native message exceeds size limit")
	ErrMalformedJSON   = errors.New("native message is not valid JSON")
)

func ReadFrame(reader io.Reader, maxSize uint32) ([]byte, error) {
	var size uint32
	if err := binary.Read(reader, binary.LittleEndian, &size); err != nil {
		return nil, err
	}
	if size == 0 || size > maxSize {
		return nil, ErrMessageTooLarge
	}
	message := make([]byte, size)
	if _, err := io.ReadFull(reader, message); err != nil {
		return nil, err
	}
	if !json.Valid(message) {
		return nil, ErrMalformedJSON
	}
	return message, nil
}

func WriteFrame(writer io.Writer, value interface{}, maxSize uint32) error {
	message, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(message) == 0 || uint64(len(message)) > uint64(maxSize) {
		return ErrMessageTooLarge
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(len(message))); err != nil {
		return err
	}
	_, err = writer.Write(message)
	return err
}

type Writer struct {
	output io.Writer
	mu     sync.Mutex
}

func NewWriter(output io.Writer) *Writer { return &Writer{output: output} }

func (w *Writer) Write(value interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return WriteFrame(w.output, value, protocol.MaxNativeMessageSize)
}

type Handler func(protocol.Request) protocol.Response

type Host struct {
	Input   io.Reader
	Writer  *Writer
	Handler Handler
	Log     io.Writer
}

func (h *Host) Serve() error {
	if h.Input == nil || h.Writer == nil || h.Handler == nil {
		return errors.New("native host is not configured")
	}
	for {
		message, err := ReadFrame(h.Input, protocol.MaxNativeMessageSize)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read native message: %w", err)
		}
		req, err := protocol.DecodeRequest(message, protocol.NativeRequestTypes)
		if err != nil {
			// A malformed envelope may not have a trustworthy request ID.
			if writeErr := h.Writer.Write(protocol.Failure("", "error", "invalid_message", err.Error())); writeErr != nil {
				return writeErr
			}
			continue
		}
		if err := h.Writer.Write(h.Handler(req)); err != nil {
			return fmt.Errorf("write native response: %w", err)
		}
	}
}

