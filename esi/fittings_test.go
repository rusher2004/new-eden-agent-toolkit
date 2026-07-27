package esi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterFittings_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"description":"PvP fit","fitting_id":123,"items":[{"flag":"HiSlot0","quantity":1,"type_id":2488}],"name":"My Fit","ship_type_id":602}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	fittings, err := client.CharacterFittings(context.Background())
	if err != nil {
		t.Fatalf("CharacterFittings returned error: %v", err)
	}
	if len(fittings) != 1 {
		t.Fatalf("fittings = %+v, want one entry", fittings)
	}
	f := fittings[0]
	if f.Description != "PvP fit" || f.FittingId != 123 || f.Name != "My Fit" || f.ShipTypeId != 602 {
		t.Errorf("fittings[0] = %+v, want Description=\"PvP fit\" FittingId=123 Name=\"My Fit\" ShipTypeId=602", f)
	}
	if len(f.Items) != 1 || f.Items[0].Flag != "HiSlot0" || f.Items[0].Quantity != 1 || f.Items[0].TypeId != 2488 {
		t.Errorf("fittings[0].Items = %+v, want one entry {Flag:\"HiSlot0\" Quantity:1 TypeId:2488}", f.Items)
	}
}

func TestCharacterFittings_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if _, err := client.CharacterFittings(context.Background()); err == nil {
		t.Fatal("CharacterFittings returned nil error, want an error for HTTP 404")
	}
}

func TestCreateCharacterFitting_SendsCorrectRequestBodyAndParsesResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"fitting_id":789}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	items := []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner{
		{Flag: "HiSlot0", Quantity: 1, TypeId: 2488},
	}

	fittingID, err := client.CreateCharacterFitting(context.Background(), "PvP fit", items, "My Fit", 602)
	if err != nil {
		t.Fatalf("CreateCharacterFitting returned error: %v", err)
	}

	var got struct {
		Description string `json:"description"`
		Items       []struct {
			Flag     string `json:"flag"`
			Quantity int64  `json:"quantity"`
			TypeId   int64  `json:"type_id"`
		} `json:"items"`
		Name       string `json:"name"`
		ShipTypeId int64  `json:"ship_type_id"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if got.Description != "PvP fit" || got.Name != "My Fit" || got.ShipTypeId != 602 {
		t.Errorf("request body = %+v, want Description=\"PvP fit\" Name=\"My Fit\" ShipTypeId=602", got)
	}
	if len(got.Items) != 1 || got.Items[0].Flag != "HiSlot0" || got.Items[0].Quantity != 1 || got.Items[0].TypeId != 2488 {
		t.Errorf("request body Items = %+v, want one entry {Flag:\"HiSlot0\" Quantity:1 TypeId:2488}", got.Items)
	}

	if fittingID != 789 {
		t.Errorf("fittingID = %d, want 789", fittingID)
	}
}

func TestDeleteCharacterFitting_SendsNoBodyAndHandlesNoContentResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.DeleteCharacterFitting(context.Background(), 123); err != nil {
		t.Fatalf("DeleteCharacterFitting returned error: %v", err)
	}
	if len(gotBody) != 0 {
		t.Errorf("request body = %q, want empty", gotBody)
	}
}

func TestDeleteCharacterFitting_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})

	if err := client.DeleteCharacterFitting(context.Background(), 123); err == nil {
		t.Fatal("DeleteCharacterFitting returned nil error, want an error for HTTP 404")
	}
}
