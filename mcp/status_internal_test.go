package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fakeStatusESI fakes get_status's one ESI call.
type fakeStatusESI struct {
	status *esiapi.Status
	err    error
}

func (f fakeStatusESI) Status(context.Context) (*esiapi.Status, error) { return f.status, f.err }

func TestGetStatusHandler_MapsFields(t *testing.T) {
	startTime := time.Date(2026, 7, 25, 11, 0, 0, 0, time.UTC)
	fake := fakeStatusESI{status: &esiapi.Status{Players: 23456, ServerVersion: "1234567", StartTime: startTime, Vip: true}}

	handler := getStatusHandler(func() statusESI { return fake })
	_, out, err := handler(context.Background(), nil, GetStatusInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.Players != 23456 || out.ServerVersion != "1234567" {
		t.Errorf("out = %+v, want Players=23456 ServerVersion=\"1234567\"", out)
	}
	if !out.StartTime.Equal(startTime) {
		t.Errorf("StartTime = %v, want %v", out.StartTime, startTime)
	}
	if !out.Vip {
		t.Errorf("Vip = %v, want true", out.Vip)
	}
}

func TestGetStatusHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")

	handler := getStatusHandler(func() statusESI { return fakeStatusESI{err: wantErr} })
	_, _, err := handler(context.Background(), nil, GetStatusInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
