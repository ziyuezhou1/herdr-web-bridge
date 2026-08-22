package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"path"
	"strings"
)

var ErrUnsafeURL = errors.New("URL scheme or host is not allowed")

func ValidateURL(raw string, allowLocalHTTP bool) (string, error) {
	if len(raw) == 0 || len(raw) > 8192 {
		return "", ErrUnsafeURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return "", ErrUnsafeURL
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.String(), nil
	case "http":
		host := strings.ToLower(u.Hostname())
		if allowLocalHTTP && (host == "localhost" || host == "127.0.0.1") {
			return u.String(), nil
		}
	}
	return "", ErrUnsafeURL
}

func NormalizeURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", ErrUnsafeURL
	}
	u.Scheme = strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port != "" && !((u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80")) {
		host = net.JoinHostPort(host, port)
	}
	u.Host = host
	u.Fragment = ""
	u.Path = path.Clean("/" + strings.TrimPrefix(u.Path, "/"))
	if u.Path == "/." {
		u.Path = "/"
	}
	return u.String(), nil
}

func SafeURLForLog(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "<invalid-url>"
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	return u.Scheme + "://" + u.Host + u.EscapedPath()
}

func HashBindingID(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:6])
}
