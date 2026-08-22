package quickactions

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/bindings"
)

func quickBinding(t *testing.T, projectPath string) bindings.Binding {
	t.Helper()
	binding, err := bindings.Create(bindings.NewBinding{
		ProjectPath: projectPath, ProjectLabel: "简历优化 / Main",
		URL: "https://example.com/task", PageTitle: "Run: #1",
		Adapter: "generic", NotificationsEnabled: true,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func TestSlugAndSafeCommand(t *testing.T) {
	if got := Slug("  VirtualDNA ChatGPT / Run #3 "); got != "virtualdna-chatgpt-run-3" {
		t.Fatalf("unexpected slug: %s", got)
	}
	binding := quickBinding(t, t.TempDir())
	command, err := PowerShellCommand(`C:\Users\O'Brien\herdr-web-bridge.exe`, binding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(command, `O''Brien`) || strings.Contains(command, binding.URL) {
		t.Fatalf("unsafe command: %s", command)
	}
}

func TestQuickActionContainsOnlyTrustedBindingReference(t *testing.T) {
	binding := quickBinding(t, t.TempDir())
	executable := filepath.Join(t.TempDir(), "herdr-web-bridge.exe")
	data, err := Render(binding, executable)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, binding.ID) || strings.Contains(content, binding.URL) {
		t.Fatalf("quick action must include the UUID and omit URL: %s", content)
	}
	if !strings.Contains(content, "generated-by") {
		t.Fatal("ownership marker missing")
	}
}

