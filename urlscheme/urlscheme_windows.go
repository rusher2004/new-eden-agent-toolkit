//go:build windows

package urlscheme

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

func register(launcherPath string) error {
	baseKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\`+Scheme, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create base registry key: %w", err)
	}
	defer baseKey.Close()

	if err := baseKey.SetStringValue("", "URL:Neat Auth Protocol"); err != nil {
		return fmt.Errorf("set protocol description: %w", err)
	}
	if err := baseKey.SetStringValue("URL Protocol", ""); err != nil {
		return fmt.Errorf("set URL Protocol marker: %w", err)
	}

	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\`+Scheme+`\shell\open\command`, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create command registry key: %w", err)
	}
	defer cmdKey.Close()

	if err := cmdKey.SetStringValue("", buildWindowsCommand(launcherPath)); err != nil {
		return fmt.Errorf("set command value: %w", err)
	}
	return nil
}
