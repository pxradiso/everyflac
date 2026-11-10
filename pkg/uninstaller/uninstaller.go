package uninstaller

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"everyflac-tui/pkg/config"
)

// Run uninstalls everyflac, cleaning binaries, shell configurations, and config files.
func Run(force bool) error {
	fmt.Println("\033[1;36m➤ everyFLAC by pxradise — Uninstaller\033[0m")
	fmt.Println("\033[38;2;86;95;137m───────────────────────────────────────────────────────────\033[0m")

	if !force {
		fmt.Println("This will remove:")
		fmt.Println("  • Executable binary (from ~/.local/bin)")
		fmt.Println("  • Desktop launcher and icon (Linux)")
		fmt.Println("  • Shell aliases and functions (Fish, Zsh, Bash)")
		fmt.Println("  • Application configuration directory")
		fmt.Print("\n\033[1;33mAre you sure you want to completely remove everyFLAC? [y/N]: \033[0m")

		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))

		if ans != "y" && ans != "yes" {
			fmt.Println("\n\033[1mUninstallation cancelled.\033[0m")
			return nil
		}
		fmt.Println()
	}

	homeDir, _ := os.UserHomeDir()

	// 1. Clean up Shell Configurations
	cleanShellConfigs(homeDir)

	// 2. Clean up App Configuration Directory
	cleanAppConfig()

	// 3. Remove Executables
	removeBinaries(homeDir)

	// 4. Remove Linux desktop launcher and icon
	cleanDesktopIntegration(homeDir)

	fmt.Println("\n\033[1;32m✨ everyFLAC has been successfully removed from your system.\033[0m")
	fmt.Println("\033[38;2;86;95;137m───────────────────────────────────────────────────────────\033[0m")
	return nil
}

func cleanShellConfigs(homeDir string) {
	// Fish function file
	fishFunc := filepath.Join(homeDir, ".config", "fish", "functions", "everyflac.fish")
	if _, err := os.Stat(fishFunc); err == nil {
		if err := os.Remove(fishFunc); err == nil {
			fmt.Printf("  \033[1;32m✔\033[0m Removed Fish function: %s\n", fishFunc)
		}
	}

	// Fish config.fish
	fishConfig := filepath.Join(homeDir, ".config", "fish", "config.fish")
	cleanFileLines(fishConfig, []string{"alias everyflac=", "# everyFLAC"})

	// If fish executable is present, erase function from memory
	if _, err := exec.LookPath("fish"); err == nil {
		_ = exec.Command("fish", "-c", "functions -e everyflac 2>/dev/null").Run()
	}

	// Zsh & Bash rc files
	cleanFileLines(filepath.Join(homeDir, ".zshrc"), []string{"alias everyflac="})
	cleanFileLines(filepath.Join(homeDir, ".bashrc"), []string{"alias everyflac="})
	cleanFileLines(filepath.Join(homeDir, ".profile"), []string{"alias everyflac="})
}

func cleanFileLines(filePath string, patterns []string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	modified := false

	for _, line := range lines {
		drop := false
		for _, pat := range patterns {
			if strings.Contains(line, pat) {
				drop = true
				modified = true
				break
			}
		}
		if !drop {
			newLines = append(newLines, line)
		}
	}

	if modified {
		// Clean trailing empty lines
		for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
			newLines = newLines[:len(newLines)-1]
		}
		newContent := strings.Join(newLines, "\n") + "\n"
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err == nil {
			fmt.Printf("  \033[1;32m✔\033[0m Cleaned aliases from: %s\n", filePath)
		}
	}
}

func cleanAppConfig() {
	cfgPath := config.ConfigPath()
	cfgDir := filepath.Dir(cfgPath)

	if strings.HasSuffix(cfgDir, "everyflac") {
		if _, err := os.Stat(cfgDir); err == nil {
			if err := os.RemoveAll(cfgDir); err == nil {
				fmt.Printf("  \033[1;32m✔\033[0m Removed configuration directory: %s\n", cfgDir)
			}
		}
	}
}

func cleanDesktopIntegration(homeDir string) {
	targets := []string{
		filepath.Join(homeDir, ".local", "share", "applications", "everyflac.desktop"),
		filepath.Join(homeDir, ".local", "share", "icons", "hicolor", "scalable", "apps", "everyflac.svg"),
		filepath.Join(homeDir, ".local", "share", "icons", "hicolor", "256x256", "apps", "everyflac.png"),
	}

	removed := false
	for _, path := range targets {
		if err := os.Remove(path); err == nil {
			removed = true
		}
	}
	if removed {
		fmt.Println("  \033[1;32m✔\033[0m Removed desktop launcher and icon")
	}
}

func removeBinaries(homeDir string) {
	targets := make(map[string]bool)

	// Currently running binary
	if currentExe, err := os.Executable(); err == nil {
		if eval, err := filepath.EvalSymlinks(currentExe); err == nil {
			targets[eval] = true
		} else {
			targets[currentExe] = true
		}
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	// Standard install directories
	installDir := filepath.Join(homeDir, ".local", "bin")
	possibleNames := []string{
		"everyflac" + ext,
		"everyflac-tui" + ext,
		"pxradise-everyflac" + ext,
	}

	for _, name := range possibleNames {
		p := filepath.Join(installDir, name)
		if _, err := os.Stat(p); err == nil {
			targets[p] = true
		}
	}

	for path := range targets {
		if _, err := os.Stat(path); err == nil {
			if err := removeExecutable(path); err == nil {
				fmt.Printf("  \033[1;32m✔\033[0m Removed executable: %s\n", path)
			} else {
				fmt.Printf("  \033[1;33m⚠\033[0m Could not immediately remove %s: %v\n", path, err)
			}
		}
	}
}
