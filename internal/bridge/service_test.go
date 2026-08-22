package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/bindings"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/buildinfo"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/herdr"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/protocol"
)

type serviceRunner struct {
	mu                sync.Mutex
	notificationCalls int
	failNotifications bool
}

func (r *serviceRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	joined := strings.Join(args, " ")
	switch {
	case joined == "workspace list":
		return []byte(`{"workspaces":[{"workspace_id":"w1","label":"VirtualDNA","focused":true,"worktree":{"checkout_path":"T:\\VirtualDNA"}}]}`), nil
	case strings.HasPrefix(joined, "workspace report-metadata"):
		return []byte(`{}`), nil
	case strings.HasPrefix(joined, "notification show"):
		r.notificationCalls++
		if r.failNotifications {
			return nil, errors.New("notification unavailable")
		}
		return []byte(`{}`), nil
	case joined == "plugin list --json":
		return []byte(`[]`), nil
	default:
		return []byte(`{}`), nil
	}
}

func TestNotificationDedupeAndFallback(t *testing.T) {
	project := filepath.Join(`T:\`, "VirtualDNA")
	binding, err := bindings.Create(bindings.NewBinding{
		ProjectPath: project, ProjectLabel: "VirtualDNA", URL: "https://chatgpt.com/c/test",
		PageTitle: "Test", Adapter: "chatgpt", NotificationsEnabled: true,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	store := bindings.NewStore(filepath.Join(t.TempDir(), "bindings.json"))
	if err := store.Add(binding); err != nil {
		t.Fatal(err)
	}
	runner := &serviceRunner{failNotifications: true}
	service := New(store, herdr.NewClient(runner), `C:\Bridge\herdr-web-bridge.exe`)
	hello := nativeRequest(t, "hello", protocol.HelloPayload{ExtensionID: buildinfo.ExtensionID})
	if response := service.HandleNative(hello); !response.OK {
		t.Fatalf("native handshake failed: %#v", response)
	}

	running := nativeRequest(t, "report_status", protocol.ReportStatusPayload{BindingID: binding.ID, State: "running", EventID: "event-1", URL: binding.URL})
	if response := service.HandleNative(running); !response.OK {
		t.Fatalf("running report failed: %#v", response)
	}
	done := nativeRequest(t, "report_status", protocol.ReportStatusPayload{BindingID: binding.ID, State: "done_unread", EventID: "event-1", URL: binding.URL})
	response := service.HandleNative(done)
	if !response.OK || !resultBool(t, response.Result, "fallbackNotification") {
		t.Fatalf("expected fallback notification: %#v", response)
	}
	response = service.HandleNative(done)
	if !response.OK || resultBool(t, response.Result, "fallbackNotification") {
		t.Fatalf("duplicate event should not notify: %#v", response)
	}
	if runner.notificationCalls != 1 {
		t.Fatalf("expected one notification call, got %d", runner.notificationCalls)
	}
}

func nativeRequest(t *testing.T, messageType string, payload interface{}) protocol.Request {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return protocol.Request{Version: 1, ID: "request-1", Type: messageType, Payload: data}
}

func resultBool(t *testing.T, value interface{}, key string) bool {
	t.Helper()
	result, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type: %T", value)
	}
	flag, _ := result[key].(bool)
	return flag
}
