// Command gentooldocs generates docs/reference/tools.md: every MCP tool neat
// registers, grouped into player-facing areas, with its description. It reads
// the tools from a live mcp.NewServer over an in-memory transport, so the
// reference can't list a tool that doesn't exist or miss one that does.
//
// It is wired via `go:generate` in mcp/server.go. Rerun `go generate ./...`
// after adding, renaming or redescribing a tool; this package's test fails
// while the committed file is stale. A tool matching no area fails generation
// until it is given one in areas below.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// area is one section of the reference. A tool belongs to the first area,
// in order, whose names list it or one of whose keywords its name contains,
// so an area must come before any whose keywords its tools also contain
// (killmails before mail; the game client's open_*_window tools first).
type area struct {
	title    string
	blurb    string
	names    []string
	keywords []string
}

var areas = []area{
	{
		title: "Characters and login",
		blurb: "Log characters in, choose what each one lets neat see, record which EVE account each is on and notes about it, and log them out.",
		names: []string{"add_character", "list_scope_groups", "list_characters", "update_character_details", "remove_character"},
	},
	{
		title:    "Game client",
		blurb:    "Act on the running EVE client: set autopilot waypoints and open in-game windows.",
		keywords: []string{"autopilot", "open_"},
	},
	{
		title: "Navigation and hauling",
		blurb: "Jump and gate routes with fuel, fatigue and dotlan links, saved routes with your notes, and cargo packed into loads, for any move: PvP, PvE or industry. Haul profiles and routes are saved in `~/.neat/haul_profiles.yaml` and `~/.neat/routes.yaml`, which you can also edit.",
		names: []string{"save_haul_profile", "list_haul_profiles", "delete_haul_profile", "save_route", "list_routes", "delete_route", "plan_jump_route", "plan_haul"},
	},
	{
		title: "Trade hubs and purchases",
		blurb: "Trade hubs to buy at, and shopping lists priced against a hub's live order book. Hub profiles are saved in `~/.neat/config.json`.",
		names: []string{"save_hub_profile", "list_hub_profiles", "quote_market_purchase"},
	},
	{
		title:    "Industry planning",
		blurb:    "Build plans, build-or-buy, job costs, research and invention, and refining and compressed ore. Facility profiles are saved in `~/.neat/facilities.yaml`, which you can also edit.",
		names:    []string{"describe_facility", "find_owned_blueprints", "check_build_skills", "check_material_stock", "solve_ore_mix"},
		keywords: []string{"_profile", "estimate_", "plan_"},
	},
	{
		title:    "Universe and static data",
		blurb:    "Items, ships, systems, regions, stations, dogma attributes and gate routes. Most of this is answered from the local SDE copy without calling ESI.",
		keywords: []string{"universe", "dogma", "get_route"},
	},
	{
		title:    "Wallet",
		blurb:    "Character and corporation wallets.",
		keywords: []string{"wallet"},
	},
	{
		title:    "Assets",
		blurb:    "What characters and corporations own, and where.",
		keywords: []string{"asset"},
	},
	{
		title:    "Market, contracts and loyalty points",
		blurb:    "Market orders and prices, contracts, insurance and LP stores.",
		keywords: []string{"market", "contract", "insurance", "loyalty"},
	},
	{
		title:    "Killmails and warfare",
		blurb:    "Killmails, wars, faction warfare, sovereignty campaigns, incursions and military campaigns.",
		keywords: []string{"killmail", "war", "fw_", "sovereignty_campaigns", "sovereignty_systems", "incursion", "military_campaign"},
	},
	{
		title:    "Mail, calendar, contacts and notifications",
		blurb:    "EVE mail, calendar events, contact lists and notifications.",
		keywords: []string{"mail", "calendar", "contact", "notification"},
	},
	{
		title:    "Fleets and fittings",
		blurb:    "Fleet management and saved ship fittings.",
		keywords: []string{"fleet", "fitting"},
	},
	{
		title:    "Activities and cosmetics",
		blurb:    "Freelance jobs, corporation projects, mercenary dens and tactical operations, skyhooks, SKINR and the Paragon Hub.",
		keywords: []string{"freelance", "mercenary", "skyhook", "skinr", "paragon", "project"},
	},
	{
		title:    "Industry, mining and planets",
		blurb:    "Blueprints, industry jobs, mining, research agents and planetary industry.",
		keywords: []string{"industry", "mining", "blueprint", "agents_research", "planet", "customs_office", "schematic", "facilities"},
	},
	{
		title:    "Corporations and alliances",
		blurb:    "Corporation and alliance details, members, roles, titles, structures and starbases. Most corporation tools need the character to hold the matching in-game roles.",
		keywords: []string{"corporation", "alliance", "access_list", "starbase", "structure"},
	},
	{
		title:    "Character details",
		blurb:    "Skills, clones, implants, location, ship, standings, medals, roles and more.",
		keywords: []string{"character"},
	},
	{
		title:    "Server and ESI status",
		blurb:    "Tranquility status and ESI's own health, changelog and compatibility dates.",
		keywords: []string{"get_status", "esi_"},
	},
}

func main() {
	out := flag.String("out", "tools.md", "output file")
	flag.Parse()
	md, err := render(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "gentooldocs:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, md, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gentooldocs: write", *out+":", err)
		os.Exit(1)
	}
}

// listTools returns every tool mcp.NewServer registers, sorted by name (the
// order the MCP SDK lists them in).
func listTools(ctx context.Context) ([]*mcpsdk.Tool, error) {
	dir, err := os.MkdirTemp("", "gentooldocs")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	server := mcp.NewServer("docs", "", config.NewStoreAt(filepath.Join(dir, "config.json")), nil)
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect server: %w", err)
	}
	// Close rather than Wait: if the client fails to connect, nothing ever
	// closes the client transport, and Wait would block forever.
	defer serverSession.Close()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "gentooldocs", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect client: %w", err)
	}
	defer session.Close()

	var tools []*mcpsdk.Tool
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, fmt.Errorf("list tools: %w", err)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// areaOf returns the index in areas of the area a tool belongs to, or -1.
func areaOf(name string) int {
	for i, a := range areas {
		if slices.Contains(a.names, name) {
			return i
		}
		for _, k := range a.keywords {
			if strings.Contains(name, k) {
				return i
			}
		}
	}
	return -1
}

func render(ctx context.Context) ([]byte, error) {
	tools, err := listTools(ctx)
	if err != nil {
		return nil, err
	}
	grouped := make([][]*mcpsdk.Tool, len(areas))
	var unplaced []string
	for _, t := range tools {
		i := areaOf(t.Name)
		if i < 0 {
			unplaced = append(unplaced, t.Name)
			continue
		}
		grouped[i] = append(grouped[i], t)
	}
	if len(unplaced) > 0 {
		return nil, fmt.Errorf("tools in no area (add them to areas in tools/gentooldocs/main.go): %s", strings.Join(unplaced, ", "))
	}

	var b bytes.Buffer
	b.WriteString("<!-- Code generated by tools/gentooldocs; DO NOT EDIT. Run `go generate ./...` to update. -->\n\n")
	b.WriteString("# neat tool reference\n\n")
	fmt.Fprintf(&b, "neat gives Claude %d tools. You don't call them yourself: ask Claude in plain language and it picks the tools it needs. This page is for when you want to know exactly what neat can do. Tools that need a character's permission list the scope group they need in their error when it's missing; see the main [README](../../README.md#choosing-what-neat-can-see).\n\n", len(tools))
	b.WriteString("## Areas\n\n")
	for i, a := range areas {
		if len(grouped[i]) == 0 {
			continue
		}
		fmt.Fprintf(&b, "- [%s](#%s) (%d)\n", a.title, anchor(a.title), len(grouped[i]))
	}
	for i, a := range areas {
		if len(grouped[i]) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n%s\n\n", a.title, a.blurb)
		for _, t := range grouped[i] {
			fmt.Fprintf(&b, "- `%s`: %s\n", t.Name, oneLine(t.Description))
		}
	}
	return b.Bytes(), nil
}

// anchor returns the GitHub heading anchor for title.
func anchor(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
