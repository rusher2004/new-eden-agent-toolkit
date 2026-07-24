package urlscheme_test

import (
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/urlscheme"
)

func TestScheme_IsEveauthPrefixed(t *testing.T) {
	if got := urlscheme.Scheme; got != "eveauth-neat" {
		t.Errorf("Scheme = %q, want %q", got, "eveauth-neat")
	}
}

func TestBuildWindowsCommand(t *testing.T) {
	got := urlscheme.ExportedBuildWindowsCommand(`C:\Users\pilot\.neat\bin\neat.exe`)
	want := `"C:\Users\pilot\.neat\bin\neat.exe" callback-handler "%1"`
	if got != want {
		t.Errorf("buildWindowsCommand = %q, want %q", got, want)
	}
}

func TestBuildDesktopFileContent(t *testing.T) {
	got := urlscheme.ExportedBuildDesktopFileContent("/home/pilot/.neat/bin/neat")
	if !strings.Contains(got, "Exec=/home/pilot/.neat/bin/neat callback-handler %u") {
		t.Errorf("desktop file content missing Exec line: %q", got)
	}
	if !strings.Contains(got, "MimeType=x-scheme-handler/eveauth-neat;") {
		t.Errorf("desktop file content missing MimeType line: %q", got)
	}
}
