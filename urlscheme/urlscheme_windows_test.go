//go:build windows

package urlscheme_test

import (
	"testing"

	"golang.org/x/sys/windows/registry"

	"github.com/rusher2004/new-eden-agent-toolkit/urlscheme"
)

func TestRegister_Windows_WritesRegistryKeys(t *testing.T) {
	const launcherPath = `C:\fake\neat.exe`

	if err := urlscheme.Register(launcherPath); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	t.Cleanup(func() {
		registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+urlscheme.Scheme+`\shell\open\command`)
		registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+urlscheme.Scheme+`\shell\open`)
		registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+urlscheme.Scheme+`\shell`)
		registry.DeleteKey(registry.CURRENT_USER, `Software\Classes\`+urlscheme.Scheme)
	})

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\`+urlscheme.Scheme+`\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("OpenKey returned error: %v", err)
	}
	defer key.Close()

	got, _, err := key.GetStringValue("")
	if err != nil {
		t.Fatalf("GetStringValue returned error: %v", err)
	}
	want := urlscheme.ExportedBuildWindowsCommand(launcherPath)
	if got != want {
		t.Errorf("command value = %q, want %q", got, want)
	}
}
