package sso

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	goesi "github.com/fnt-eve/goesi-openapi"
)

// ScopeGroup is a named set of ESI scopes a user can grant a character, in
// terms of what they'll use neat for rather than individual scopes.
type ScopeGroup struct {
	Name        string
	Description string
	Scopes      []string
}

// DefaultScopeGroup is always requested, whatever else is: basic character
// information most tools need.
const DefaultScopeGroup = "basic"

// ScopeGroups lists every group neat can request. Logins request the
// default group plus whichever the user picks, instead of every scope at
// once: a login requesting all of them pushed the cookies CCP's login site
// sets past its request-header limit after a couple of logins in a row
// ("HTTP Error 400. The size of the request headers is too long").
//
// Every scope an ESI endpoint requires (esi/scopes_gen.go) must be in one
// group; esi's tests enforce it. The new-style scopes (esi.activity.char:read,
// esi.cosmetic.char:read) must also be enabled on the ESI application at
// developers.eveonline.com, or CCP rejects any login requesting them.
var ScopeGroups = []ScopeGroup{
	{DefaultScopeGroup, "Basic character info: skills and skill queue, location, current ship, online status, clones and implants.", []string{
		"publicData",
		"esi-skills.read_skills.v1",
		"esi-skills.read_skillqueue.v1",
		"esi-location.read_location.v1",
		"esi-location.read_ship_type.v1",
		"esi-location.read_online.v1",
		"esi-clones.read_clones.v1",
		"esi-clones.read_implants.v1",
	}},
	{"character", "More character details: standings, jump fatigue, loyalty points, medals, titles, corporation roles, research agents, notifications and faction warfare stats.", []string{
		"esi-characters.read_standings.v1",
		"esi-characters.read_fatigue.v1",
		"esi-characters.read_loyalty.v1",
		"esi-characters.read_medals.v1",
		"esi-characters.read_titles.v1",
		"esi-characters.read_corporation_roles.v1",
		"esi-characters.read_agents_research.v1",
		"esi-characters.read_notifications.v1",
		"esi-characters.read_fw_stats.v1",
	}},
	{"wallet", "Wallet balance, journal and transactions.", []string{
		"esi-wallet.read_character_wallet.v1",
	}},
	{"assets", "Assets: what the character owns and where.", []string{
		"esi-assets.read_assets.v1",
	}},
	{"industry", "Industry: blueprints, industry jobs and mining ledger. Industry planning also wants assets and market.", []string{
		"esi-characters.read_blueprints.v1",
		"esi-industry.read_character_jobs.v1",
		"esi-industry.read_character_mining.v1",
	}},
	{"market", "Market: the character's orders and order history, and market orders in player structures.", []string{
		"esi-markets.read_character_orders.v1",
		"esi-markets.structure_markets.v1",
	}},
	{"contracts", "The character's contracts, bids and contract items.", []string{
		"esi-contracts.read_character_contracts.v1",
	}},
	{"mail", "EVE mail: read, send, label and delete mail.", []string{
		"esi-mail.read_mail.v1",
		"esi-mail.send_mail.v1",
		"esi-mail.organize_mail.v1",
	}},
	{"calendar", "Calendar: read events and respond to them.", []string{
		"esi-calendar.read_calendar_events.v1",
		"esi-calendar.respond_calendar_events.v1",
	}},
	{"contacts", "Contacts: read, add, edit and remove the character's contacts.", []string{
		"esi-characters.read_contacts.v1",
		"esi-characters.write_contacts.v1",
	}},
	{"fittings", "Saved ship fittings: read, save and delete.", []string{
		"esi-fittings.read_fittings.v1",
		"esi-fittings.write_fittings.v1",
	}},
	{"fleet", "Fleets: read and manage the character's fleet.", []string{
		"esi-fleets.read_fleet.v1",
		"esi-fleets.write_fleet.v1",
	}},
	{"ui", "Control the game client: set autopilot waypoints and open in-game windows.", []string{
		"esi-ui.write_waypoint.v1",
		"esi-ui.open_window.v1",
	}},
	{"killmails", "The character's recent killmails.", []string{
		"esi-killmails.read_killmails.v1",
	}},
	{"planets", "Planetary industry colonies.", []string{
		"esi-planets.manage_planets.v1",
	}},
	{"structures", "Player structures: look up and search structures, structure access lists, and the character's mercenary dens.", []string{
		"esi-universe.read_structures.v1",
		"esi-search.search_structures.v1",
		"esi-access.read_lists.v1",
		"esi-structures.read_character.v1",
	}},
	{"activities", "Activities: freelance jobs, mercenary tactical operations and military campaign objectives.", []string{
		"esi-activities.read_character.v1",
		"esi-characters.read_freelance_jobs.v1",
		"esi.activity.char:read",
	}},
	{"cosmetics", "SKINR licenses and Paragon Hub listings.", []string{
		"esi.cosmetic.char:read",
	}},
	{"corporation", "Corporation management (needs in-game corporation roles): members, member tracking, roles, titles, divisions, standings, contacts, medals, faction warfare stats, killmails and alliance contacts.", []string{
		"esi-corporations.read_corporation_membership.v1",
		"esi-corporations.track_members.v1",
		"esi-corporations.read_titles.v1",
		"esi-corporations.read_divisions.v1",
		"esi-corporations.read_standings.v1",
		"esi-corporations.read_contacts.v1",
		"esi-corporations.read_medals.v1",
		"esi-corporations.read_fw_stats.v1",
		"esi-killmails.read_corporation_killmails.v1",
		"esi-alliances.read_contacts.v1",
	}},
	{"corporation_finance", "Corporation finances (needs in-game corporation roles): wallets, assets, contracts, market orders and container logs.", []string{
		"esi-wallet.read_corporation_wallets.v1",
		"esi-assets.read_corporation_assets.v1",
		"esi-contracts.read_corporation_contracts.v1",
		"esi-markets.read_corporation_orders.v1",
		"esi-corporations.read_container_logs.v1",
	}},
	{"corporation_industry", "Corporation industry and infrastructure (needs in-game corporation roles): jobs, blueprints, mining, customs offices, facilities, structures, starbases, skyhooks and sovereignty hubs, projects and freelance jobs.", []string{
		"esi-industry.read_corporation_jobs.v1",
		"esi-corporations.read_blueprints.v1",
		"esi-industry.read_corporation_mining.v1",
		"esi-planets.read_customs_offices.v1",
		"esi-corporations.read_facilities.v1",
		"esi-corporations.read_structures.v1",
		"esi-corporations.read_starbases.v1",
		"esi-structures.read_corporation.v1",
		"esi-corporations.read_projects.v1",
		"esi-corporations.read_freelance_jobs.v1",
	}},
}

// ScopeGroupFor returns the name of the group that holds scope, or "" if
// none does.
func ScopeGroupFor(scope string) string {
	for _, g := range ScopeGroups {
		if slices.Contains(g.Scopes, scope) {
			return g.Name
		}
	}
	return ""
}

// ScopesFor returns the scopes of the default group plus groups, sorted and
// without duplicates. It fails on an unknown group name.
func ScopesFor(groups []string) ([]string, error) {
	set := map[string]bool{}
	for _, name := range append([]string{DefaultScopeGroup}, groups...) {
		g, ok := scopeGroup(name)
		if !ok {
			return nil, fmt.Errorf("unknown scope group %q; valid groups: %s", name, strings.Join(ScopeGroupNames(), ", "))
		}
		for _, s := range g.Scopes {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out, nil
}

// GrantedGroups returns the names of the groups whose scopes are all in
// granted, in ScopeGroups order.
func GrantedGroups(granted []string) []string {
	var out []string
	for _, g := range ScopeGroups {
		all := true
		for _, s := range g.Scopes {
			if !slices.Contains(granted, s) {
				all = false
				break
			}
		}
		if all {
			out = append(out, g.Name)
		}
	}
	return out
}

// ScopeGroupNames lists every group name, in ScopeGroups order.
func ScopeGroupNames() []string {
	names := make([]string, len(ScopeGroups))
	for i, g := range ScopeGroups {
		names[i] = g.Name
	}
	return names
}

func scopeGroup(name string) (ScopeGroup, bool) {
	for _, g := range ScopeGroups {
		if g.Name == name {
			return g, true
		}
	}
	return ScopeGroup{}, false
}

// GrantedScopes returns the scopes characterID's current access token
// carries, read from the token's scp claim. The token was verified when
// the character logged in and comes straight from CCP's token endpoint
// over TLS, so it is decoded here without re-verifying its signature.
func GrantedScopes(ctx context.Context, clientID string, characterID int32, userAgent string) ([]string, error) {
	if _, err := AuthenticatedClient(ctx, clientID, characterID, userAgent); err != nil {
		return nil, err
	}
	clientCacheMu.Lock()
	entry, ok := clientCache[characterID]
	clientCacheMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("character %d was removed while reading its scopes", characterID)
	}
	tok, err := entry.source.Token()
	if err != nil {
		return nil, fmt.Errorf("get access token for character %d: %w", characterID, err)
	}
	return scopesFromAccessToken(tok.AccessToken)
}

func scopesFromAccessToken(accessToken string) ([]string, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("access token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode access token payload: %w", err)
	}
	var claims goesi.ESIJWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse access token claims: %w", err)
	}
	return []string(claims.Scopes), nil
}

// ExportedScopesFromAccessToken exposes scopesFromAccessToken for external tests.
func ExportedScopesFromAccessToken(accessToken string) ([]string, error) {
	return scopesFromAccessToken(accessToken)
}
