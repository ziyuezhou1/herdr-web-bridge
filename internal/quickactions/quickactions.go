// Package quickactions creates project-local Herdr Plus actions owned by this bridge.
package quickactions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/bindings"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/fileutil"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

var ErrNotBridgeOwned = errors.New("quick action is not owned by herdr-web-bridge")
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func Slug(input string) string {
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(input) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if (unicode.IsSpace(r) || r == '-' || r == '_') && out.Len() > 0 && !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	value := strings.Trim(out.String(), "-")
	if value == "" {
		value = "web"
	}
	if len(value) > 40 {
		value = strings.Trim(value[:40], "-")
	}
	return value
}

func FilePath(binding bindings.Binding) (string, error) {
	projectPath, err := security.NormalizeWindowsPath(binding.ProjectPath)
	if err != nil {
		return "", err
	}
	shortID := strings.ReplaceAll(binding.ID, "-", "")
	if len(shortID) < 8 {
		return "", errors.New("invalid binding id")
	}
	name := fmt.Sprintf("open-web-%s-%s.toml", Slug(binding.ProjectLabel+"-"+binding.PageTitle), shortID[:8])
	return filepath.Join(projectPath, ".herdr-plus", "quick-actions", name), nil
}

func PowerShellCommand(executablePath, bindingID string) (string, error) {
	if !filepath.IsAbs(executablePath) || !uuidPattern.MatchString(bindingID) {
		return "", errors.New("invalid executable path or binding id")
	}
	escape := func(value string) string { return strings.ReplaceAll(value, "'", "''") }
	return fmt.Sprintf("& '%s' open --binding '%s'", escape(filepath.Clean(executablePath)), escape(bindingID)), nil
}

func Render(binding bindings.Binding, executablePath string) ([]byte, error) {
	command, err := PowerShellCommand(executablePath, binding.ID)
	if err != nil {
		return nil, err
	}
	name := security.TruncateRunes("打开网页："+binding.ProjectLabel+" / "+binding.PageTitle, 80)
	description := "聚焦已打开的 Edge 标签页；若不存在则重新打开绑定网址"
	content := fmt.Sprintf("# generated-by = \"herdr-web-bridge:%s\"\nname = %s\ndescription = %s\ncommand = %s\n", binding.ID, strconv.Quote(name), strconv.Quote(description), strconv.Quote(command))
	return []byte(content), nil
}

func Write(binding bindings.Binding, executablePath string) (string, error) {
	path, err := FilePath(binding)
	if err != nil {
		return "", err
	}
	data, err := Render(binding, executablePath)
	if err != nil {
		return "", err
	}
	if err := fileutil.WriteAtomic(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func Remove(binding bindings.Binding) error {
	path := binding.QuickActionFile
	if path == "" {
		var err error
		path, err = FilePath(binding)
		if err != nil {
			return err
		}
	}
	expectedRoot := filepath.Join(binding.ProjectPath, ".herdr-plus", "quick-actions")
	relative, err := filepath.Rel(expectedRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return ErrNotBridgeOwned
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	marker := fmt.Sprintf("# generated-by = \"herdr-web-bridge:%s\"", binding.ID)
	if !strings.Contains(string(data), marker) {
		return ErrNotBridgeOwned
	}
	return os.Remove(path)
}
