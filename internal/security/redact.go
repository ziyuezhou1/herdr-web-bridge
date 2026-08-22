package security

import (
	"regexp"
	"strings"
)

var headerSecretPattern = regexp.MustCompile(`(?i)(authorization|cookie)(\s*[=:]\s*)[^\r\n]+`)
var secretPattern = regexp.MustCompile(`(?i)(token|password|secret)(\s*[=:]\s*)([^\s,;]+)`)
var urlPattern = regexp.MustCompile(`https?://[^\s"']+`)

func RedactLog(input string) string {
	redacted := headerSecretPattern.ReplaceAllString(input, `$1$2<redacted>`)
	redacted = secretPattern.ReplaceAllString(redacted, `$1$2<redacted>`)
	return urlPattern.ReplaceAllStringFunc(redacted, SafeURLForLog)
}

func TruncateRunes(input string, max int) string {
	runes := []rune(strings.Join(strings.Fields(input), " "))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
