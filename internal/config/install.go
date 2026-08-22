// Package config reads installer-owned, non-secret bridge settings.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/buildinfo"
)

var extensionIDPattern = regexp.MustCompile(`^[a-p]{32}$`)

type Install struct {
	SchemaVersion int    `json:"schemaVersion"`
	ExtensionID  string `json:"extensionId"`
	HostName     string `json:"hostName"`
	ExecutablePath string `json:"executablePath,omitempty"`
}

func DefaultInstall() Install {
	return Install{SchemaVersion: 1, ExtensionID: buildinfo.ExtensionID, HostName: buildinfo.HostName}
}

func Path() (string, error) {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return filepath.Join(root, "HerdrWebBridge", "install.json"), nil
}

func Load() (Install, error) {
	defaults := DefaultInstall()
	path, err := Path()
	if err != nil {
		return defaults, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, err
	}
	var installed Install
	if err := json.Unmarshal(data, &installed); err != nil {
		return defaults, err
	}
	if installed.SchemaVersion != 1 || !extensionIDPattern.MatchString(installed.ExtensionID) || installed.HostName != buildinfo.HostName {
		return defaults, errors.New("invalid install configuration")
	}
	if installed.ExecutablePath != "" && (!filepath.IsAbs(installed.ExecutablePath) || filepath.Base(installed.ExecutablePath) != "herdr-web-bridge.exe") {
		return defaults, errors.New("invalid installed executable path")
	}
	return installed, nil
}
