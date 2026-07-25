package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// addCharacter is overridable in tests so no real browser/network/OAuth2
// flow runs during unit tests.
var addCharacter = sso.AddCharacter

// allScopes is the full ESI scope list neat currently requests at login.
// Deliberately broad during development so any ESI endpoint can be
// exercised while domain tools are being built; narrowing this to a
// minimal per-tool set is a follow-up once real tools exist and their
// actual scope needs are known.
var allScopes = []string{
	"publicData",
	"esi-calendar.respond_calendar_events.v1",
	"esi-calendar.read_calendar_events.v1",
	"esi-location.read_location.v1",
	"esi-location.read_ship_type.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-skills.read_skills.v1",
	"esi-skills.read_skillqueue.v1",
	"esi-wallet.read_corporation_wallet.v1",
	"esi-search.search_structures.v1",
	"esi-clones.read_clones.v1",
	"esi-characters.read_contacts.v1",
	"esi-killmails.read_killmails.v1",
	"esi-corporations.read_corporation_membership.v1",
	"esi-assets.read_assets.v1",
	"esi-planets.manage_planets.v1",
	"esi-fleets.read_fleet.v1",
	"esi-fleets.write_fleet.v1",
	"esi-ui.open_window.v1",
	"esi-characters.write_contacts.v1",
	"esi-fittings.read_fittings.v1",
	"esi-fittings.write_fittings.v1",
	"esi-markets.structure_markets.v1",
	"esi-corporations.read_structures.v1",
	"esi-characters.read_loyalty.v1",
	"esi-characters.read_chat_channels.v1",
	"esi-characters.read_medals.v1",
	"esi-characters.read_standings.v1",
	"esi-characters.read_agents_research.v1",
	"esi-industry.read_character_jobs.v1",
	"esi-markets.read_character_orders.v1",
	"esi-characters.read_blueprints.v1",
	"esi-characters.read_corporation_roles.v1",
	"esi-location.read_online.v1",
	"esi-contracts.read_character_contracts.v1",
	"esi-clones.read_implants.v1",
	"esi-characters.read_fatigue.v1",
	"esi-killmails.read_corporation_killmails.v1",
	"esi-corporations.track_members.v1",
	"esi-wallet.read_corporation_wallets.v1",
	"esi-characters.read_notifications.v1",
	"esi-corporations.read_divisions.v1",
	"esi-corporations.read_contacts.v1",
	"esi-assets.read_corporation_assets.v1",
	"esi-corporations.read_titles.v1",
	"esi-corporations.read_blueprints.v1",
	"esi-corporations.read_standings.v1",
	"esi-industry.read_corporation_jobs.v1",
	"esi-markets.read_corporation_orders.v1",
	"esi-corporations.read_container_logs.v1",
	"esi-industry.read_character_mining.v1",
	"esi-industry.read_corporation_mining.v1",
	"esi-planets.read_customs_offices.v1",
	"esi-corporations.read_facilities.v1",
	"esi-corporations.read_medals.v1",
	"esi-characters.read_titles.v1",
	"esi-alliances.read_contacts.v1",
	"esi-corporations.read_fw_stats.v1",
	"esi-corporations.read_projects.v1",
	"esi-corporations.read_freelance_jobs.v1",
	"esi-characters.read_freelance_jobs.v1",
	"esi-structures.read_corporation.v1",
	"esi-structures.read_character.v1",
	"esi-activities.read_character.v1",
	"esi-access.read_lists.v1",
	"esi-wallet.read_character_wallet.v1",
	"esi-universe.read_structures.v1",
	"esi-ui.write_waypoint.v1",
	"esi-contracts.read_corporation_contracts.v1",
	"esi-corporations.read_starbases.v1",
	"esi-characters.read_fw_stats.v1",
}

// AddCharacterOutput is add_character's tool result.
type AddCharacterOutput struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
}

func addCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[any, AddCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, AddCharacterOutput, error) {
		char, err := addCharacter(ctx, clientID, sso.UserAgent(version), allScopes)
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("add character: %w", err)
		}

		cfg, err := store.Load()
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("load config: %w", err)
		}
		upsertCharacter(&cfg, config.Character{ID: char.ID, Name: char.Name})
		if err := store.Save(cfg); err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("save config: %w", err)
		}

		return nil, AddCharacterOutput{CharacterID: char.ID, CharacterName: char.Name}, nil
	}
}

// upsertCharacter replaces the entry in cfg.Characters matching c.ID, or
// appends c if no existing entry matches.
func upsertCharacter(cfg *config.Config, c config.Character) {
	for i, existing := range cfg.Characters {
		if existing.ID == c.ID {
			cfg.Characters[i] = c
			return
		}
	}
	cfg.Characters = append(cfg.Characters, c)
}
