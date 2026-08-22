//go:build !windows

package fileutil

import "os"

func Replace(source, target string) error { return os.Rename(source, target) }

