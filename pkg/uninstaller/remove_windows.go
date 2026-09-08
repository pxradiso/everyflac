//go:build windows

package uninstaller

import (
	"fmt"
	"os"
	"os/exec"
)

func removeExecutable(path string) error {
	if err := os.Remove(path); err == nil {
		return nil
	}
	// If locked by the running process, schedule delayed deletion after exit
	cmd := exec.Command("cmd.exe", "/C", fmt.Sprintf("ping 127.0.0.1 -n 2 >nul & del /F /Q \"%s\"", path))
	return cmd.Start()
}
