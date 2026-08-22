package security

import (
	"strings"
	"testing"
)

func TestNormalizeWindowsPathAndDriveCase(t *testing.T) {
	got, err := NormalizeWindowsPath(`t:\Work\Project\..\Project\`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `T:\Work\Project` {
		t.Fatalf("unexpected normalized path: %q", got)
	}
	if !SameWindowsPath(`t:\WORK\Project`, `T:\work\project\`) {
		fatal := "drive and path comparison must be case-insensitive on Windows"
		t.Fatal(fatal)
	}
}

func TestURLSchemeValidation(t *testing.T) {
	allowed := []string{"https://chatgpt.com/c/abc?secret=value", "http://localhost:3000/test", "http://127.0.0.1/tool"}
	for _, value := range allowed {
		if _, err := ValidateURL(value, true); err != nil {
			t.Errorf("expected %s to pass: %v", value, err)
		}
	}
	blocked := []string{"http://example.com", "file:///C:/secret", "javascript:alert(1)", "data:text/plain,x", "ftp://example.com/a", "shell:AppsFolder", "custom://open"}
	for _, value := range blocked {
		if _, err := ValidateURL(value, true); err == nil {
			t.Errorf("expected %s to be rejected", value)
		}
	}
}

func TestLogRedaction(t *testing.T) {
	input := "Authorization: Bearer header-token\ntoken=abc https://example.com/path?q=secret#fragment"
	got := RedactLog(input)
	for _, secret := range []string{"header-token", "abc", "q=secret", "fragment"} {
		if strings.Contains(got, secret) {
			t.Fatalf("redaction leaked %q in %q", secret, got)
		}
	}
	if !strings.Contains(got, "https://example.com/path") {
		t.Fatalf("safe URL origin/path missing: %s", got)
	}
}
