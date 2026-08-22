package protocol

import (
	"errors"
	"testing"
)

func TestDecodeRequestRejectsUnknownAndTrailingFields(t *testing.T) {
	_, err := DecodeRequest([]byte(`{"version":1,"id":"1","type":"run_shell"}`), IPCRequestTypes)
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected unsupported type, got %v", err)
	}
	_, err = DecodeRequest([]byte(`{"version":1,"id":"1","type":"ping","command":"calc.exe"}`), IPCRequestTypes)
	if !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("expected invalid envelope, got %v", err)
	}
}

func TestIPCWhitelist(t *testing.T) {
	allowed := []string{"open_binding", "list_bindings", "ping", "status"}
	if len(IPCRequestTypes) != len(allowed) {
		t.Fatalf("IPC allowlist changed: %#v", IPCRequestTypes)
	}
	for _, name := range allowed {
		if _, ok := IPCRequestTypes[name]; !ok {
			t.Fatalf("missing allowed IPC command %s", name)
		}
	}
}
