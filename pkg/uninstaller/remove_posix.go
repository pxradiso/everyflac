//go:build !windows

package uninstaller

import "os"

func removeExecutable(path string) error {
	return os.Remove(path)
}
