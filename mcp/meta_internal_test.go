package mcp

import (
	"context"
	"errors"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fakeMetaESI fakes the meta tools' ESI calls; err is returned by every call.
type fakeMetaESI struct {
	changelog *esiapi.MetaChangelog
	dates     *esiapi.MetaCompatibilityDates
	name      *esiapi.MetaName
	status    *esiapi.MetaStatus
	err       error
}

func (f fakeMetaESI) MetaChangelog(context.Context) (*esiapi.MetaChangelog, error) {
	return f.changelog, f.err
}

func (f fakeMetaESI) MetaCompatibilityDates(context.Context) (*esiapi.MetaCompatibilityDates, error) {
	return f.dates, f.err
}

func (f fakeMetaESI) MetaName(context.Context) (*esiapi.MetaName, error) { return f.name, f.err }

func (f fakeMetaESI) MetaStatus(context.Context) (*esiapi.MetaStatus, error) {
	return f.status, f.err
}

func openMetaFake(c metaESI) func() metaESI { return func() metaESI { return c } }

func TestGetEsiChangelogHandler_MapsFields(t *testing.T) {
	fake := fakeMetaESI{changelog: &esiapi.MetaChangelog{Changelog: map[string][]esiapi.MetaChangelogEntry{
		"2026-07-01": {{CompatibilityDate: "2026-07-01", Description: "Added a thing", Method: "GET", Path: "/foo", Type: "added"}},
	}}}

	handler := getEsiChangelogHandler(openMetaFake(fake))
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
	fake := fakeMetaESI{dates: &esiapi.MetaCompatibilityDates{CompatibilityDates: []string{"2026-01-01", "2026-07-01"}}}

	handler := listEsiCompatibilityDatesHandler(openMetaFake(fake))
	_, out, err := handler(context.Background(), nil, ListEsiCompatibilityDatesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.CompatibilityDates) != 2 || out.CompatibilityDates[0] != "2026-01-01" || out.CompatibilityDates[1] != "2026-07-01" {
		t.Errorf("out.CompatibilityDates = %+v, want [2026-01-01 2026-07-01]", out.CompatibilityDates)
	}
}

func TestGetEsiNameHandler_MapsFields(t *testing.T) {
	fake := fakeMetaESI{name: &esiapi.MetaName{Current: "ESI", History: []esiapi.MetaNameEntry{{Date: "2016-01-01", Name: "CREST"}}}}

	handler := getEsiNameHandler(openMetaFake(fake))
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
	fake := fakeMetaESI{status: &esiapi.MetaStatus{Routes: []esiapi.MetaStatusRoute{{Method: "GET", Path: "/characters/{character_id}/", Status: "green"}}}}

	handler := getEsiStatusHandler(openMetaFake(fake))
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
	wantErr := errors.New("upstream failure")

	handler := getEsiChangelogHandler(openMetaFake(fakeMetaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, GetEsiChangelogInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListEsiCompatibilityDatesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := listEsiCompatibilityDatesHandler(openMetaFake(fakeMetaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, ListEsiCompatibilityDatesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetEsiNameHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := getEsiNameHandler(openMetaFake(fakeMetaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, GetEsiNameInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetEsiStatusHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")

	handler := getEsiStatusHandler(openMetaFake(fakeMetaESI{err: wantErr}))
	_, _, err := handler(context.Background(), nil, GetEsiStatusInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
