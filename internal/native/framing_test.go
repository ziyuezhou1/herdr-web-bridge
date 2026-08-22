package native

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buffer bytes.Buffer
	want := map[string]interface{}{"version": 1, "type": "ping", "ok": true}
	if err := WriteFrame(&buffer, want, 1024); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrame(&buffer, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"ok":true,"type":"ping","version":1}` {
		t.Fatalf("unexpected JSON: %s", got)
	}
}

func TestMalformedJSON(t *testing.T) {
	var buffer bytes.Buffer
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(4))
	buffer.WriteString("nope")
	_, err := ReadFrame(&buffer, 1024)
	if !errors.Is(err, ErrMalformedJSON) {
		t.Fatalf("expected malformed JSON, got %v", err)
	}
}

func TestOversizedFrameRejectedBeforeAllocation(t *testing.T) {
	var buffer bytes.Buffer
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(65<<10))
	_, err := ReadFrame(&buffer, 64<<10)
	if !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("expected size error, got %v", err)
	}
}
