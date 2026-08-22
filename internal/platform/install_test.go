package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHerdrExecutableFallsBackToPerUserInstall(t *testing.T) {
	localAppData := t.TempDir()
	executable := filepath.Join(localAppData, "Programs", "Herdr", "bin", "herdr.exe")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	t.Setenv("LOCALAPPDATA", localAppData)

	got, err := HerdrExecutable()
	if err != nil {
		t.Fatalf("HerdrExecutable() error = %v", err)
	}
	if got != executable {
		t.Fatalf("HerdrExecutable() = %q, want %q", got, executable)
	}
}
