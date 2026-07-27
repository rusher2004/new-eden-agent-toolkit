package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_esi_changelog (public) ---

type ChangelogEntry struct {
	CompatibilityDate string `json:"compatibilityDate"`
	Description       string `json:"description"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Type              string `json:"type"`
}

var getEsiChangelog = func(ctx context.Context, userAgent string) (map[string][]ChangelogEntry, error) {
	client := esi.NewPublicClient(userAgent)
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

func getEsiChangelogHandler(version string) mcpsdk.ToolHandlerFor[GetEsiChangelogInput, GetEsiChangelogOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiChangelogInput) (*mcpsdk.CallToolResult, GetEsiChangelogOutput, error) {
		changelog, err := getEsiChangelog(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetEsiChangelogOutput{}, fmt.Errorf("get esi changelog: %w", err)
		}
		return nil, GetEsiChangelogOutput{Changelog: changelog}, nil
	}
}

// --- list_esi_compatibility_dates (public, no input) ---

var listEsiCompatibilityDates = func(ctx context.Context, userAgent string) ([]string, error) {
	client := esi.NewPublicClient(userAgent)
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

func listEsiCompatibilityDatesHandler(version string) mcpsdk.ToolHandlerFor[ListEsiCompatibilityDatesInput, ListEsiCompatibilityDatesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListEsiCompatibilityDatesInput) (*mcpsdk.CallToolResult, ListEsiCompatibilityDatesOutput, error) {
		dates, err := listEsiCompatibilityDates(ctx, sso.UserAgent(version))
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

var getEsiName = func(ctx context.Context, userAgent string) (string, []NameEntry, error) {
	client := esi.NewPublicClient(userAgent)
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

func getEsiNameHandler(version string) mcpsdk.ToolHandlerFor[GetEsiNameInput, GetEsiNameOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiNameInput) (*mcpsdk.CallToolResult, GetEsiNameOutput, error) {
		current, history, err := getEsiName(ctx, sso.UserAgent(version))
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

var getEsiStatus = func(ctx context.Context, userAgent string) ([]RouteStatus, error) {
	client := esi.NewPublicClient(userAgent)
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

func getEsiStatusHandler(version string) mcpsdk.ToolHandlerFor[GetEsiStatusInput, GetEsiStatusOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetEsiStatusInput) (*mcpsdk.CallToolResult, GetEsiStatusOutput, error) {
		routes, err := getEsiStatus(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetEsiStatusOutput{}, fmt.Errorf("get esi status: %w", err)
		}
		return nil, GetEsiStatusOutput{Routes: routes}, nil
	}
}
