package commands_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

func TestVersionCommand_PrintsVersion(t *testing.T) {
	var out bytes.Buffer
	root := commands.Root("1.2.3", "test-client-id")
	root.Writer = &out

	if err := root.Run(context.Background(), []string{"neat", "version"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "1.2.3" {
		t.Errorf("output = %q, want %q", got, "1.2.3")
	}
}
