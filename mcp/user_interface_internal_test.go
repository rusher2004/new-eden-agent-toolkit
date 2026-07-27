package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// --- set_autopilot_waypoint ---

func TestSetAutopilotWaypointHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := setAutopilotWaypoint
	t.Cleanup(func() { setAutopilotWaypoint = orig })
	var gotCharacterID int32
	var gotDestinationID int64
	var gotAddToBeginning, gotClearOtherWaypoints bool
	setAutopilotWaypoint = func(ctx context.Context, clientID string, characterID int32, userAgent string, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
		gotCharacterID, gotDestinationID, gotAddToBeginning, gotClearOtherWaypoints = characterID, destinationID, addToBeginning, clearOtherWaypoints
		return nil
	}

	handler := setAutopilotWaypointHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, SetAutopilotWaypointInput{
		CharacterID: 7, DestinationID: 30000142, AddToBeginning: true, ClearOtherWaypoints: false,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCharacterID != 7 || gotDestinationID != 30000142 || !gotAddToBeginning || gotClearOtherWaypoints {
		t.Errorf("captured args = CharacterID=%d DestinationID=%d AddToBeginning=%v ClearOtherWaypoints=%v, want 7/30000142/true/false", gotCharacterID, gotDestinationID, gotAddToBeginning, gotClearOtherWaypoints)
	}
	if out.CharacterID != 7 || out.DestinationID != 30000142 || !out.AddToBeginning || out.ClearOtherWaypoints {
		t.Errorf("out = %+v, want CharacterID=7 DestinationID=30000142 AddToBeginning=true ClearOtherWaypoints=false", out)
	}
}

func TestSetAutopilotWaypointHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := setAutopilotWaypoint
	t.Cleanup(func() { setAutopilotWaypoint = orig })
	setAutopilotWaypoint = func(ctx context.Context, clientID string, characterID int32, userAgent string, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
		t.Fatal("setAutopilotWaypoint should not be called for an unknown character")
		return nil
	}

	handler := setAutopilotWaypointHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, SetAutopilotWaypointInput{CharacterID: 999, DestinationID: 30000142})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestSetAutopilotWaypointHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := setAutopilotWaypoint
	t.Cleanup(func() { setAutopilotWaypoint = orig })
	wantErr := errors.New("upstream failure")
	setAutopilotWaypoint = func(ctx context.Context, clientID string, characterID int32, userAgent string, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
		return wantErr
	}

	handler := setAutopilotWaypointHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, SetAutopilotWaypointInput{CharacterID: 7, DestinationID: 30000142})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- open_contract_window ---

func TestOpenContractWindowHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openContractWindow
	t.Cleanup(func() { openContractWindow = orig })
	var gotCharacterID int32
	var gotContractID int64
	openContractWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) error {
		gotCharacterID, gotContractID = characterID, contractID
		return nil
	}

	handler := openContractWindowHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, OpenContractWindowInput{CharacterID: 7, ContractID: 5001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCharacterID != 7 || gotContractID != 5001 {
		t.Errorf("captured args = CharacterID=%d ContractID=%d, want 7/5001", gotCharacterID, gotContractID)
	}
	if out.CharacterID != 7 || out.ContractID != 5001 {
		t.Errorf("out = %+v, want CharacterID=7 ContractID=5001", out)
	}
}

func TestOpenContractWindowHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := openContractWindow
	t.Cleanup(func() { openContractWindow = orig })
	openContractWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) error {
		t.Fatal("openContractWindow should not be called for an unknown character")
		return nil
	}

	handler := openContractWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenContractWindowInput{CharacterID: 999, ContractID: 5001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestOpenContractWindowHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openContractWindow
	t.Cleanup(func() { openContractWindow = orig })
	wantErr := errors.New("upstream failure")
	openContractWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) error {
		return wantErr
	}

	handler := openContractWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenContractWindowInput{CharacterID: 7, ContractID: 5001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- open_information_window ---

func TestOpenInformationWindowHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openInformationWindow
	t.Cleanup(func() { openInformationWindow = orig })
	var gotCharacterID int32
	var gotTargetID int64
	openInformationWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetID int64) error {
		gotCharacterID, gotTargetID = characterID, targetID
		return nil
	}

	handler := openInformationWindowHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, OpenInformationWindowInput{CharacterID: 7, TargetID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCharacterID != 7 || gotTargetID != 98000001 {
		t.Errorf("captured args = CharacterID=%d TargetID=%d, want 7/98000001", gotCharacterID, gotTargetID)
	}
	if out.CharacterID != 7 || out.TargetID != 98000001 {
		t.Errorf("out = %+v, want CharacterID=7 TargetID=98000001", out)
	}
}

func TestOpenInformationWindowHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := openInformationWindow
	t.Cleanup(func() { openInformationWindow = orig })
	openInformationWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetID int64) error {
		t.Fatal("openInformationWindow should not be called for an unknown character")
		return nil
	}

	handler := openInformationWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenInformationWindowInput{CharacterID: 999, TargetID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestOpenInformationWindowHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openInformationWindow
	t.Cleanup(func() { openInformationWindow = orig })
	wantErr := errors.New("upstream failure")
	openInformationWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetID int64) error {
		return wantErr
	}

	handler := openInformationWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenInformationWindowInput{CharacterID: 7, TargetID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- open_market_details_window ---

func TestOpenMarketDetailsWindowHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openMarketDetailsWindow
	t.Cleanup(func() { openMarketDetailsWindow = orig })
	var gotCharacterID int32
	var gotTypeID int64
	openMarketDetailsWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, typeID int64) error {
		gotCharacterID, gotTypeID = characterID, typeID
		return nil
	}

	handler := openMarketDetailsWindowHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, OpenMarketDetailsWindowInput{CharacterID: 7, TypeID: 587})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCharacterID != 7 || gotTypeID != 587 {
		t.Errorf("captured args = CharacterID=%d TypeID=%d, want 7/587", gotCharacterID, gotTypeID)
	}
	if out.CharacterID != 7 || out.TypeID != 587 {
		t.Errorf("out = %+v, want CharacterID=7 TypeID=587", out)
	}
}

func TestOpenMarketDetailsWindowHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := openMarketDetailsWindow
	t.Cleanup(func() { openMarketDetailsWindow = orig })
	openMarketDetailsWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, typeID int64) error {
		t.Fatal("openMarketDetailsWindow should not be called for an unknown character")
		return nil
	}

	handler := openMarketDetailsWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenMarketDetailsWindowInput{CharacterID: 999, TypeID: 587})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestOpenMarketDetailsWindowHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openMarketDetailsWindow
	t.Cleanup(func() { openMarketDetailsWindow = orig })
	wantErr := errors.New("upstream failure")
	openMarketDetailsWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, typeID int64) error {
		return wantErr
	}

	handler := openMarketDetailsWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenMarketDetailsWindowInput{CharacterID: 7, TypeID: 587})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- open_new_mail_window ---

func TestOpenNewMailWindowHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openNewMailWindow
	t.Cleanup(func() { openNewMailWindow = orig })
	var gotCharacterID int32
	var gotRecipients []int64
	var gotSubject, gotBody string
	var gotToCorpOrAllianceID, gotToMailingListID *int64
	openNewMailWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		gotCharacterID, gotRecipients, gotSubject, gotBody = characterID, recipients, subject, body
		gotToCorpOrAllianceID, gotToMailingListID = toCorpOrAllianceID, toMailingListID
		return nil
	}

	toCorpOrAllianceID := int64(1000035)
	handler := openNewMailWindowHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, OpenNewMailWindowInput{
		CharacterID: 7, Recipients: []int64{12345}, Subject: "Subject", Body: "Body",
		ToCorpOrAllianceID: &toCorpOrAllianceID,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotCharacterID != 7 || gotSubject != "Subject" || gotBody != "Body" || len(gotRecipients) != 1 || gotRecipients[0] != 12345 {
		t.Errorf("captured args = CharacterID=%d Subject=%q Body=%q Recipients=%v, want 7/Subject/Body/[12345]", gotCharacterID, gotSubject, gotBody, gotRecipients)
	}
	if gotToCorpOrAllianceID == nil || *gotToCorpOrAllianceID != 1000035 {
		t.Errorf("captured ToCorpOrAllianceID = %v, want 1000035", gotToCorpOrAllianceID)
	}
	if gotToMailingListID != nil {
		t.Errorf("captured ToMailingListID = %v, want nil", gotToMailingListID)
	}
	if out.CharacterID != 7 || out.Subject != "Subject" || out.Body != "Body" || len(out.Recipients) != 1 || out.Recipients[0] != 12345 {
		t.Errorf("out = %+v, want CharacterID=7 Subject=Subject Body=Body Recipients=[12345]", out)
	}
	if out.ToCorpOrAllianceID == nil || *out.ToCorpOrAllianceID != 1000035 {
		t.Errorf("ToCorpOrAllianceID = %v, want 1000035", out.ToCorpOrAllianceID)
	}
	if out.ToMailingListID != nil {
		t.Errorf("ToMailingListID = %v, want nil", out.ToMailingListID)
	}
}

func TestOpenNewMailWindowHandler_RejectsBothMailingGroupFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openNewMailWindow
	t.Cleanup(func() { openNewMailWindow = orig })
	openNewMailWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		t.Fatal("openNewMailWindow should not be called when both mailing-group fields are set")
		return nil
	}

	corpID := int64(1000035)
	listID := int64(2)
	handler := openNewMailWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenNewMailWindowInput{
		CharacterID: 7, Recipients: []int64{12345}, Subject: "Subject", Body: "Body",
		ToCorpOrAllianceID: &corpID, ToMailingListID: &listID,
	})
	if err == nil {
		t.Fatal("expected error when both toCorpOrAllianceId and toMailingListId are set")
	}
}

func TestOpenNewMailWindowHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := openNewMailWindow
	t.Cleanup(func() { openNewMailWindow = orig })
	openNewMailWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		t.Fatal("openNewMailWindow should not be called for an unknown character")
		return nil
	}

	handler := openNewMailWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenNewMailWindowInput{CharacterID: 999, Recipients: []int64{12345}, Subject: "Subject", Body: "Body"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestOpenNewMailWindowHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := openNewMailWindow
	t.Cleanup(func() { openNewMailWindow = orig })
	wantErr := errors.New("upstream failure")
	openNewMailWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		return wantErr
	}

	handler := openNewMailWindowHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, OpenNewMailWindowInput{CharacterID: 7, Recipients: []int64{12345}, Subject: "Subject", Body: "Body"})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
