package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// addCharacters is overridable in tests so no real browser/network/OAuth2
// flow runs during unit tests.
var addCharacters = sso.AddCharacters

// allScopes is the full ESI scope list neat currently requests at login.
// Deliberately broad during development so any ESI endpoint can be
// exercised while domain tools are being built; narrowing this to a
// minimal per-tool set is a follow-up once real tools exist and their
// actual scope needs are known.
//
// Verified against CCP's live OpenAPI 3.0 spec (fetched with the current compatibility date, per
// docs/superpowers/specs/2026-07-25-esi-coverage-roadmap-design.md's fetch method) that every
// scope every implemented tool actually needs is present here. Two scopes that were present but
// don't correspond to any real CCP scope were removed as part of that verification:
// "esi-wallet.read_corporation_wallet.v1" (singular -- a bogus duplicate of the real
// "esi-wallet.read_corporation_wallets.v1", plural, below) and
// "esi-characters.read_chat_channels.v1" (no corresponding real scope or tool at all -- chat
// channels aren't a feature this project implements).
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
	// New-style scope names (namespace.resource.subject:verb rather than esi-x.y.v1), introduced
	// with ESI compatibility date 2026-08-18 and declared in the live spec's OAuth2 scheme.
	// esi.activity.char:read gates the Military Campaigns tag's 2 character endpoints;
	// esi.cosmetic.char:read gates Cosmetics' 2 character endpoints and Paragon Hub's 4
	// authenticated ones. The ESI application registered at developers.eveonline.com must have
	// both enabled too, or CCP's SSO rejects the whole login with invalid_scope.
	"esi.activity.char:read",
	"esi.cosmetic.char:read",
}

// AddedCharacter is one character added by add_character.
type AddedCharacter struct {
	CharacterID   int32  `json:"characterId"`
	CharacterName string `json:"characterName"`
}

// AddCharacterOutput is add_character's tool result: every character the
// user logged in during the browser session, in the order they were added.
type AddCharacterOutput struct {
	Characters []AddedCharacter `json:"characters"`
}

func addCharacterHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[any, AddCharacterOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ any) (*mcpsdk.CallToolResult, AddCharacterOutput, error) {
		chars, err := addCharacters(ctx, clientID, sso.UserAgent(version), sso.AddOptions{
			Scopes: allScopes,
			// Save each character as its login completes, not once the
			// session ends, so the session page lists it and an abandoned
			// session doesn't lose it.
			OnAdded: func(c sso.Character) error {
				cfg, err := store.Load()
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				upsertCharacter(&cfg, config.Character{ID: c.ID, Name: c.Name})
				if err := store.Save(cfg); err != nil {
					return fmt.Errorf("save config: %w", err)
				}
				return nil
			},
			SavedCharacters: func() ([]sso.Character, error) {
				cfg, err := store.Load()
				if err != nil {
					return nil, fmt.Errorf("load config: %w", err)
				}
				out := make([]sso.Character, len(cfg.Characters))
				for i, c := range cfg.Characters {
					out[i] = sso.Character{ID: c.ID, Name: c.Name}
				}
				return out, nil
			},
		})
		if err != nil {
			return nil, AddCharacterOutput{}, fmt.Errorf("add character: %w", err)
		}

		out := AddCharacterOutput{Characters: make([]AddedCharacter, len(chars))}
		for i, c := range chars {
			out.Characters[i] = AddedCharacter{CharacterID: c.ID, CharacterName: c.Name}
		}
		return nil, out, nil
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
