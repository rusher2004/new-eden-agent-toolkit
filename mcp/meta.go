package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// metaESI is the ESI calls the get_esi_* and list_esi_compatibility_dates tools make; all are public.
type metaESI interface {
	MetaChangelog(ctx context.Context) (*esiapi.MetaChangelog, error)
	MetaCompatibilityDates(ctx context.Context) (*esiapi.MetaCompatibilityDates, error)
	MetaName(ctx context.Context) (*esiapi.MetaName, error)
	MetaStatus(ctx context.Context) (*esiapi.MetaStatus, error)
}

// --- get_esi_changelog (public) ---

type ChangelogEntry struct {
	CompatibilityDate string `json:"compatibilityDate"`
	Description       string `json:"description"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Type              string `json:"type"`
}

func getEsiChangelog(ctx context.Context, client metaESI) (map[string][]ChangelogEntry, error) {
	changelog, err := client.MetaChangelog(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]ChangelogEntry, len(changelog.Changelog))
	for date, entries := range changelog.Changelog {
		converted := make([]ChangelogEntry, len(entries))
		for i, e := range entries {
			converted[i] = ChangelogEntry{
				CompatibilityDate: e.CompatibilityDate, Description: e.Description,
				Method: e.Method, Path: e.Path, Type: e.Type,
			}
		}
		out[date] = converted
	}
	return out, nil
}

type GetEsiChangelogInput struct{}
type GetEsiChangelogOutput struct {
	Changelog map[string][]ChangelogEntry `json:"changelog"`
}

// AddGetEsiChangelog registers get_esi_changelog. open builds the ESI client; any client with the
// metaESI methods will do.
func AddGetEsiChangelog[C metaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_esi_changelog",
		Description: "Get ESI's own changelog, keyed by date (public data, no character required).",
	}, getEsiChangelogHandler(open))
}

func getEsiChangelogHandler[C metaESI](open func() C) mcpsdk.ToolHandlerFor[GetEsiChangelogInput, GetEsiChangelogOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiChangelogInput) (*mcpsdk.CallToolResult, GetEsiChangelogOutput, error) {
		changelog, err := getEsiChangelog(ctx, open())
		if err != nil {
			return nil, GetEsiChangelogOutput{}, fmt.Errorf("get esi changelog: %w", err)
		}
		return nil, GetEsiChangelogOutput{Changelog: changelog}, nil
	}
}

// --- list_esi_compatibility_dates (public, no input) ---

func listEsiCompatibilityDates(ctx context.Context, client metaESI) ([]string, error) {
	dates, err := client.MetaCompatibilityDates(ctx)
	if err != nil {
		return nil, err
	}
	return dates.CompatibilityDates, nil
}

type ListEsiCompatibilityDatesInput struct{}
type ListEsiCompatibilityDatesOutput struct {
	CompatibilityDates []string `json:"compatibilityDates"`
}

// AddListEsiCompatibilityDates registers list_esi_compatibility_dates. open builds the ESI client; any client with the
// metaESI methods will do.
func AddListEsiCompatibilityDates[C metaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_esi_compatibility_dates",
		Description: "List every compatibility date ESI currently accepts (public data, no character required).",
	}, listEsiCompatibilityDatesHandler(open))
}

func listEsiCompatibilityDatesHandler[C metaESI](open func() C) mcpsdk.ToolHandlerFor[ListEsiCompatibilityDatesInput, ListEsiCompatibilityDatesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListEsiCompatibilityDatesInput) (*mcpsdk.CallToolResult, ListEsiCompatibilityDatesOutput, error) {
		dates, err := listEsiCompatibilityDates(ctx, open())
		if err != nil {
			return nil, ListEsiCompatibilityDatesOutput{}, fmt.Errorf("list esi compatibility dates: %w", err)
		}
		return nil, ListEsiCompatibilityDatesOutput{CompatibilityDates: dates}, nil
	}
}

// --- get_esi_name (public, no input) ---

type NameEntry struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

func getEsiName(ctx context.Context, client metaESI) (string, []NameEntry, error) {
	name, err := client.MetaName(ctx)
	if err != nil {
		return "", nil, err
	}
	history := make([]NameEntry, len(name.History))
	for i, h := range name.History {
		history[i] = NameEntry{Date: h.Date, Name: h.Name}
	}
	return name.Current, history, nil
}

type GetEsiNameInput struct{}
type GetEsiNameOutput struct {
	Current string      `json:"current"`
	History []NameEntry `json:"history"`
}

// AddGetEsiName registers get_esi_name. open builds the ESI client; any client with the
// metaESI methods will do.
func AddGetEsiName[C metaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_esi_name",
		Description: "Get ESI's current and historical names (public data, no character required).",
	}, getEsiNameHandler(open))
}

func getEsiNameHandler[C metaESI](open func() C) mcpsdk.ToolHandlerFor[GetEsiNameInput, GetEsiNameOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiNameInput) (*mcpsdk.CallToolResult, GetEsiNameOutput, error) {
		current, history, err := getEsiName(ctx, open())
		if err != nil {
			return nil, GetEsiNameOutput{}, fmt.Errorf("get esi name: %w", err)
		}
		return nil, GetEsiNameOutput{Current: current, History: history}, nil
	}
}

// --- get_esi_status (public, no input) ---

type RouteStatus struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

func getEsiStatus(ctx context.Context, client metaESI) ([]RouteStatus, error) {
	status, err := client.MetaStatus(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RouteStatus, len(status.Routes))
	for i, r := range status.Routes {
		out[i] = RouteStatus{Method: r.Method, Path: r.Path, Status: r.Status}
	}
	return out, nil
}

type GetEsiStatusInput struct{}
type GetEsiStatusOutput struct {
	Routes []RouteStatus `json:"routes"`
}

// AddGetEsiStatus registers get_esi_status. open builds the ESI client; any client with the
// metaESI methods will do.
func AddGetEsiStatus[C metaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_esi_status",
		Description: "Get the health status of every ESI route (public data, no character required).",
	}, getEsiStatusHandler(open))
}

func getEsiStatusHandler[C metaESI](open func() C) mcpsdk.ToolHandlerFor[GetEsiStatusInput, GetEsiStatusOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiStatusInput) (*mcpsdk.CallToolResult, GetEsiStatusOutput, error) {
		routes, err := getEsiStatus(ctx, open())
		if err != nil {
			return nil, GetEsiStatusOutput{}, fmt.Errorf("get esi status: %w", err)
		}
		return nil, GetEsiStatusOutput{Routes: routes}, nil
	}
}
