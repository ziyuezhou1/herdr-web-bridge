package security

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var ErrInvalidProjectPath = errors.New("project path must be an absolute directory path")

func NormalizeWindowsPath(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", ErrInvalidProjectPath
	}
	clean := filepath.Clean(input)
	if !filepath.IsAbs(clean) {
		return "", ErrInvalidProjectPath
	}
	volume := filepath.VolumeName(clean)
	if len(volume) == 2 && volume[1] == ':' {
		clean = strings.ToUpper(volume[:1]) + clean[1:]
	}
	if clean != filepath.VolumeName(clean)+string(filepath.Separator) {
		clean = strings.TrimRight(clean, `\/`)
	}
	return clean, nil
}

func ValidateExistingDirectory(input string) (string, error) {
	normalized, err := NormalizeWindowsPath(input)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(normalized)
	if err != nil || !info.IsDir() {
		return "", ErrInvalidProjectPath
	}
	return normalized, nil
}

func SameWindowsPath(a, b string) bool {
	left, errA := NormalizeWindowsPath(a)
	right, errB := NormalizeWindowsPath(b)
	return errA == nil && errB == nil && strings.EqualFold(left, right)
}

