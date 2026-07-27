package mcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGetStatusHandler_MapsFields(t *testing.T) {
	orig := getStatus
	t.Cleanup(func() { getStatus = orig })
	startTime := time.Date(2026, 7, 25, 11, 0, 0, 0, time.UTC)
	vip := false
	getStatus = func(ctx context.Context, userAgent string) (int32, string, time.Time, *bool, error) {
		return 23456, "1234567", startTime, &vip, nil
	}

	handler := getStatusHandler("v1")
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
	if out.Vip == nil || *out.Vip {
		t.Errorf("Vip = %v, want *false", out.Vip)
	}
}

func TestGetStatusHandler_PropagatesError(t *testing.T) {
	orig := getStatus
	t.Cleanup(func() { getStatus = orig })
	getStatus = func(ctx context.Context, userAgent string) (int32, string, time.Time, *bool, error) {
		return 0, "", time.Time{}, nil, errors.New("boom")
	}

	handler := getStatusHandler("v1")
	_, _, err := handler(context.Background(), nil, GetStatusInput{})
	if err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
