//go:build !windows

package platform

import "errors"

func OpenURL(string) error { return errors.New("URL fallback is supported only on Windows") }

