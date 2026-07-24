//go:build linux

package urlscheme

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopFileName = "neat-url-handler.desktop"

func register(launcherPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	appsDir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		return fmt.Errorf("create applications directory: %w", err)
	}

	desktopFilePath := filepath.Join(appsDir, desktopFileName)
	content := buildDesktopFileContent(launcherPath)
	if err := os.WriteFile(desktopFilePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write .desktop file: %w", err)
	}

	if err := commandRunner("xdg-mime", "default", desktopFileName, "x-scheme-handler/"+Scheme); err != nil {
		return fmt.Errorf("run xdg-mime: %w", err)
	}
	return nil
}
