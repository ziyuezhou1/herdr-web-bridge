package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/buildinfo"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

type NativeHostStatus struct {
	Registered bool   `json:"registered"`
	Details    string `json:"details"`
}

func NativeHostRegistration() NativeHostStatus {
	key := `HKCU\Software\Microsoft\Edge\NativeMessagingHosts\` + buildinfo.HostName
	command := exec.Command("reg.exe", "query", key, "/ve")
	output, err := command.CombinedOutput()
	if err != nil {
		return NativeHostStatus{Registered: false, Details: "current-user Edge native host registration not found"}
	}
	return NativeHostStatus{Registered: true, Details: security.TruncateRunes(security.RedactLog(string(output)), 200)}
}

func InstalledExecutable() string {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		return ""
	}
	return filepath.Join(root, "Programs", "HerdrWebBridge", "herdr-web-bridge.exe")
}

func HerdrExecutable() (string, error) {
	if path, err := exec.LookPath("herdr"); err == nil && strings.TrimSpace(path) != "" {
		return path, nil
	}
	for _, candidate := range herdrFallbackPaths(os.Getenv("LOCALAPPDATA")) {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("Herdr CLI was not found on PATH or in the standard per-user install directory")
}

func herdrFallbackPaths(localAppData string) []string {
	if strings.TrimSpace(localAppData) == "" {
		return nil
	}
	return []string{filepath.Join(localAppData, "Programs", "Herdr", "bin", "herdr.exe")}
}
