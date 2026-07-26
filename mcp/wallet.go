package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// getWalletBalance is overridable in tests so no real network/auth runs.
var getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.WalletBalance(ctx)
}

// GetWalletBalanceInput is get_wallet_balance's tool input.
type GetWalletBalanceInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// GetWalletBalanceOutput is get_wallet_balance's tool result.
type GetWalletBalanceOutput struct {
	CharacterID int32   `json:"characterId"`
	BalanceISK  float64 `json:"balanceIsk"`
}

func getWalletBalanceHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetWalletBalanceInput, GetWalletBalanceOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetWalletBalanceInput) (*mcpsdk.CallToolResult, GetWalletBalanceOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("get wallet balance for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		balance, err := getWalletBalance(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, GetWalletBalanceOutput{}, fmt.Errorf("get wallet balance: %w", err)
		}
		return nil, GetWalletBalanceOutput{CharacterID: in.CharacterID, BalanceISK: balance}, nil
	}
}
