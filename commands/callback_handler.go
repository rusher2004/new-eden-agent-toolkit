package commands

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"
	"github.com/rusher2004/new-eden-agent-toolkit/urlscheme"
)

func callbackHandlerCommand() *cli.Command {
	return &cli.Command{
		Name:   "callback-handler",
		Usage:  "internal: delivers an OS URL-scheme activation to a waiting add_character call",
		Hidden: true,
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) != 1 {
				return errors.New("callback-handler requires exactly one argument: the activation URL")
			}

			// Windows' shell\open\command substitutes %1 with the raw
			// activation URL with no escaping, ahead of our process even
			// starting - a URL containing a `"` can inject extra argv
			// tokens (the well-known URL-protocol-handler command-injection
			// pattern). We can't fix that substitution from inside our own
			// process (it already happened), but we can refuse to forward
			// anything that isn't a well-formed eveauth-neat:// URL before
			// it ever reaches the callback socket - this, plus the
			// exactly-one-argument check above, is the actual mitigation
			// boundary for this class of attack.
			rawURL := args[0]
			parsed, err := url.Parse(rawURL)
			if err != nil {
				return fmt.Errorf("parse activation URL: %w", err)
			}
			if parsed.Scheme != urlscheme.Scheme {
				return fmt.Errorf("activation URL has unexpected scheme %q, want %q", parsed.Scheme, urlscheme.Scheme)
			}

			if err := sso.DeliverCallback(rawURL); err != nil {
				return fmt.Errorf("deliver callback: %w", err)
			}
			return nil
		},
	}
}
