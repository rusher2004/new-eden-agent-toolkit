package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestGetEsiChangelogHandler_MapsFields(t *testing.T) {
	orig := getEsiChangelog
	t.Cleanup(func() { getEsiChangelog = orig })
	getEsiChangelog = func(ctx context.Context, userAgent string) (map[string][]ChangelogEntry, error) {
		return map[string][]ChangelogEntry{
			"2026-07-01": {{CompatibilityDate: "2026-07-01", Description: "Added a thing", Method: "GET", Path: "/foo", Type: "added"}},
		}, nil
	}

	handler := getEsiChangelogHandler("v1")
	_, out, err := handler(context.Background(), nil, GetEsiChangelogInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	entries, ok := out.Changelog["2026-07-01"]
	if !ok || len(entries) != 1 {
		t.Fatalf("out.Changelog[2026-07-01] = %+v, want one entry", entries)
	}
	e := entries[0]
	if e.CompatibilityDate != "2026-07-01" || e.Description != "Added a thing" || e.Method != "GET" || e.Path != "/foo" || e.Type != "added" {
		t.Errorf("entry = %+v, want CompatibilityDate=2026-07-01 Description=\"Added a thing\" Method=GET Path=/foo Type=added", e)
	}
}

func TestListEsiCompatibilityDatesHandler_MapsFields(t *testing.T) {
	orig := listEsiCompatibilityDates
	t.Cleanup(func() { listEsiCompatibilityDates = orig })
	listEsiCompatibilityDates = func(ctx context.Context, userAgent string) ([]string, error) {
		return []string{"2026-01-01", "2026-07-01"}, nil
	}

	handler := listEsiCompatibilityDatesHandler("v1")
	_, out, err := handler(context.Background(), nil, ListEsiCompatibilityDatesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.CompatibilityDates) != 2 || out.CompatibilityDates[0] != "2026-01-01" || out.CompatibilityDates[1] != "2026-07-01" {
		t.Errorf("out.CompatibilityDates = %+v, want [2026-01-01 2026-07-01]", out.CompatibilityDates)
	}
}

func TestGetEsiNameHandler_MapsFields(t *testing.T) {
	orig := getEsiName
	t.Cleanup(func() { getEsiName = orig })
	getEsiName = func(ctx context.Context, userAgent string) (string, []NameEntry, error) {
		return "ESI", []NameEntry{{Date: "2016-01-01", Name: "CREST"}}, nil
	}

	handler := getEsiNameHandler("v1")
	_, out, err := handler(context.Background(), nil, GetEsiNameInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Current != "ESI" {
		t.Errorf("out.Current = %q, want \"ESI\"", out.Current)
	}
	if len(out.History) != 1 || out.History[0].Date != "2016-01-01" || out.History[0].Name != "CREST" {
		t.Errorf("out.History = %+v, want one entry {Date:2016-01-01 Name:CREST}", out.History)
	}
}

func TestGetEsiStatusHandler_MapsFields(t *testing.T) {
	orig := getEsiStatus
	t.Cleanup(func() { getEsiStatus = orig })
	getEsiStatus = func(ctx context.Context, userAgent string) ([]RouteStatus, error) {
		return []RouteStatus{{Method: "GET", Path: "/characters/{character_id}/", Status: "green"}}, nil
	}

	handler := getEsiStatusHandler("v1")
	_, out, err := handler(context.Background(), nil, GetEsiStatusInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Routes) != 1 {
		t.Fatalf("out.Routes = %+v, want one entry", out.Routes)
	}
	r := out.Routes[0]
	if r.Method != "GET" || r.Path != "/characters/{character_id}/" || r.Status != "green" {
		t.Errorf("out.Routes[0] = %+v, want Method=GET Path=/characters/{character_id}/ Status=green", r)
	}
}

func TestGetEsiChangelogHandler_PropagatesError(t *testing.T) {
	orig := getEsiChangelog
	t.Cleanup(func() { getEsiChangelog = orig })
	wantErr := errors.New("upstream failure")
	getEsiChangelog = func(ctx context.Context, userAgent string) (map[string][]ChangelogEntry, error) {
		return nil, wantErr
	}

	handler := getEsiChangelogHandler("v1")
	_, _, err := handler(context.Background(), nil, GetEsiChangelogInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListEsiCompatibilityDatesHandler_PropagatesError(t *testing.T) {
	orig := listEsiCompatibilityDates
	t.Cleanup(func() { listEsiCompatibilityDates = orig })
	wantErr := errors.New("upstream failure")
	listEsiCompatibilityDates = func(ctx context.Context, userAgent string) ([]string, error) {
		return nil, wantErr
	}

	handler := listEsiCompatibilityDatesHandler("v1")
	_, _, err := handler(context.Background(), nil, ListEsiCompatibilityDatesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetEsiNameHandler_PropagatesError(t *testing.T) {
	orig := getEsiName
	t.Cleanup(func() { getEsiName = orig })
	wantErr := errors.New("upstream failure")
	getEsiName = func(ctx context.Context, userAgent string) (string, []NameEntry, error) {
		return "", nil, wantErr
	}

	handler := getEsiNameHandler("v1")
	_, _, err := handler(context.Background(), nil, GetEsiNameInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetEsiStatusHandler_PropagatesError(t *testing.T) {
	orig := getEsiStatus
	t.Cleanup(func() { getEsiStatus = orig })
	wantErr := errors.New("upstream failure")
	getEsiStatus = func(ctx context.Context, userAgent string) ([]RouteStatus, error) {
		return nil, wantErr
	}

	handler := getEsiStatusHandler("v1")
	_, _, err := handler(context.Background(), nil, GetEsiStatusInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
