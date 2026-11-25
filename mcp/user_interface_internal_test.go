package mcp

import (
	"context"
	"errors"
	"strings"
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

	fake := &userInterfaceFake{}
	var gotDestinationID int64
	var gotAddToBeginning, gotClearOtherWaypoints bool
	fake.setAutopilotWaypoint = func(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
		gotDestinationID, gotAddToBeginning, gotClearOtherWaypoints = destinationID, addToBeginning, clearOtherWaypoints
		return nil
	}

	handler := setAutopilotWaypointHandler(fake.open, store)
	_, out, err := handler(context.Background(), nil, SetAutopilotWaypointInput{
		CharacterID: 7, DestinationID: 30000142, AddToBeginning: true, ClearOtherWaypoints: false,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(fake.openedFor) != 1 || fake.openedFor[0] != 7 || gotDestinationID != 30000142 || !gotAddToBeginning || gotClearOtherWaypoints {
		t.Errorf("captured args = openedFor=%v DestinationID=%d AddToBeginning=%v ClearOtherWaypoints=%v, want 7/30000142/true/false", fake.openedFor, gotDestinationID, gotAddToBeginning, gotClearOtherWaypoints)
	}
	if out.CharacterID != 7 || out.DestinationID != 30000142 || !out.AddToBeginning || out.ClearOtherWaypoints {
		t.Errorf("out = %+v, want CharacterID=7 DestinationID=30000142 AddToBeginning=true ClearOtherWaypoints=false", out)
	}
}

func TestSetAutopilotWaypointHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &userInterfaceFake{}
	fake.setAutopilotWaypoint = func(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
		t.Fatal("setAutopilotWaypoint should not be called for an unknown character")
		return nil
	}

	handler := setAutopilotWaypointHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	wantErr := errors.New("upstream failure")
	fake.setAutopilotWaypoint = func(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
		return wantErr
	}

	handler := setAutopilotWaypointHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	var gotContractID int64
	fake.openContractWindow = func(ctx context.Context, contractID int64) error {
		gotContractID = contractID
		return nil
	}

	handler := openContractWindowHandler(fake.open, store)
	_, out, err := handler(context.Background(), nil, OpenContractWindowInput{CharacterID: 7, ContractID: 5001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(fake.openedFor) != 1 || fake.openedFor[0] != 7 || gotContractID != 5001 {
		t.Errorf("captured args = openedFor=%v ContractID=%d, want 7/5001", fake.openedFor, gotContractID)
	}
	if out.CharacterID != 7 || out.ContractID != 5001 {
		t.Errorf("out = %+v, want CharacterID=7 ContractID=5001", out)
	}
}

func TestOpenContractWindowHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &userInterfaceFake{}
	fake.openContractWindow = func(ctx context.Context, contractID int64) error {
		t.Fatal("openContractWindow should not be called for an unknown character")
		return nil
	}

	handler := openContractWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	wantErr := errors.New("upstream failure")
	fake.openContractWindow = func(ctx context.Context, contractID int64) error {
		return wantErr
	}

	handler := openContractWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	var gotTargetID int64
	fake.openInformationWindow = func(ctx context.Context, targetID int64) error {
		gotTargetID = targetID
		return nil
	}

	handler := openInformationWindowHandler(fake.open, store)
	_, out, err := handler(context.Background(), nil, OpenInformationWindowInput{CharacterID: 7, TargetID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(fake.openedFor) != 1 || fake.openedFor[0] != 7 || gotTargetID != 98000001 {
		t.Errorf("captured args = openedFor=%v TargetID=%d, want 7/98000001", fake.openedFor, gotTargetID)
	}
	if out.CharacterID != 7 || out.TargetID != 98000001 {
		t.Errorf("out = %+v, want CharacterID=7 TargetID=98000001", out)
	}
}

func TestOpenInformationWindowHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &userInterfaceFake{}
	fake.openInformationWindow = func(ctx context.Context, targetID int64) error {
		t.Fatal("openInformationWindow should not be called for an unknown character")
		return nil
	}

	handler := openInformationWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	wantErr := errors.New("upstream failure")
	fake.openInformationWindow = func(ctx context.Context, targetID int64) error {
		return wantErr
	}

	handler := openInformationWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	var gotTypeID int64
	fake.openMarketDetailsWindow = func(ctx context.Context, typeID int64) error {
		gotTypeID = typeID
		return nil
	}

	handler := openMarketDetailsWindowHandler(fake.open, store)
	_, out, err := handler(context.Background(), nil, OpenMarketDetailsWindowInput{CharacterID: 7, TypeID: 587})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(fake.openedFor) != 1 || fake.openedFor[0] != 7 || gotTypeID != 587 {
		t.Errorf("captured args = openedFor=%v TypeID=%d, want 7/587", fake.openedFor, gotTypeID)
	}
	if out.CharacterID != 7 || out.TypeID != 587 {
		t.Errorf("out = %+v, want CharacterID=7 TypeID=587", out)
	}
}

func TestOpenMarketDetailsWindowHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	fake := &userInterfaceFake{}
	fake.openMarketDetailsWindow = func(ctx context.Context, typeID int64) error {
		t.Fatal("openMarketDetailsWindow should not be called for an unknown character")
		return nil
	}

	handler := openMarketDetailsWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	wantErr := errors.New("upstream failure")
	fake.openMarketDetailsWindow = func(ctx context.Context, typeID int64) error {
		return wantErr
	}

	handler := openMarketDetailsWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	var gotRecipients []int64
	var gotSubject, gotBody string
	var gotToCorpOrAllianceID, gotToMailingListID *int64
	fake.openNewMailWindow = func(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		gotRecipients, gotSubject, gotBody = recipients, subject, body
		gotToCorpOrAllianceID, gotToMailingListID = toCorpOrAllianceID, toMailingListID
		return nil
	}

	toCorpOrAllianceID := int64(1000035)
	handler := openNewMailWindowHandler(fake.open, store)
	_, out, err := handler(context.Background(), nil, OpenNewMailWindowInput{
		CharacterID: 7, Recipients: []int64{12345}, Subject: "Subject", Body: "Body",
		ToCorpOrAllianceID: &toCorpOrAllianceID,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(fake.openedFor) != 1 || fake.openedFor[0] != 7 || gotSubject != "Subject" || gotBody != "Body" || len(gotRecipients) != 1 || gotRecipients[0] != 12345 {
		t.Errorf("captured args = openedFor=%v Subject=%q Body=%q Recipients=%v, want 7/Subject/Body/[12345]", fake.openedFor, gotSubject, gotBody, gotRecipients)
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

	fake := &userInterfaceFake{}
	fake.openNewMailWindow = func(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		t.Fatal("openNewMailWindow should not be called when both mailing-group fields are set")
		return nil
	}

	corpID := int64(1000035)
	listID := int64(2)
	handler := openNewMailWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	fake.openNewMailWindow = func(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		t.Fatal("openNewMailWindow should not be called for an unknown character")
		return nil
	}

	handler := openNewMailWindowHandler(fake.open, store)
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

	fake := &userInterfaceFake{}
	wantErr := errors.New("upstream failure")
	fake.openNewMailWindow = func(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
		return wantErr
	}

	handler := openNewMailWindowHandler(fake.open, store)
	_, _, err := handler(context.Background(), nil, OpenNewMailWindowInput{CharacterID: 7, Recipients: []int64{12345}, Subject: "Subject", Body: "Body"})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// TestUserInterfaceHandlers_OpenFailure: every handler that opens an ESI client wraps and returns
// the opener's error, and never reaches an ESI call.
func TestUserInterfaceHandlers_OpenFailure(t *testing.T) {
	store := newTestStoreWithCharacter(t)
	openErr := errors.New("keychain locked")
	fake := &userInterfaceFake{openErr: openErr}
	tests := []struct {
		name string
		call func() error
	}{
		{"set_autopilot_waypoint", func() error {
			return callHandlerForOpenFailure(setAutopilotWaypointHandler(fake.open, store), SetAutopilotWaypointInput{CharacterID: 7})
		}},
		{"open_contract_window", func() error {
			return callHandlerForOpenFailure(openContractWindowHandler(fake.open, store), OpenContractWindowInput{CharacterID: 7})
		}},
		{"open_information_window", func() error {
			return callHandlerForOpenFailure(openInformationWindowHandler(fake.open, store), OpenInformationWindowInput{CharacterID: 7})
		}},
		{"open_market_details_window", func() error {
			return callHandlerForOpenFailure(openMarketDetailsWindowHandler(fake.open, store), OpenMarketDetailsWindowInput{CharacterID: 7})
		}},
		{"open_new_mail_window", func() error {
			return callHandlerForOpenFailure(openNewMailWindowHandler(fake.open, store), OpenNewMailWindowInput{CharacterID: 7})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if !errors.Is(err, openErr) {
				t.Fatalf("err = %v, want wrapping %v", err, openErr)
			}
			if !strings.Contains(err.Error(), strings.ReplaceAll(tt.name, "_", " ")) {
				t.Errorf("err = %q, want it to start with the tool name %q", err, tt.name)
			}
		})
	}
}

// userInterfaceFake fakes userInterfaceESI: set a field to answer that call, and open records the character IDs it was asked for.
type userInterfaceFake struct {
	setAutopilotWaypoint    func(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error
	openContractWindow      func(ctx context.Context, contractID int64) error
	openInformationWindow   func(ctx context.Context, targetID int64) error
	openMarketDetailsWindow func(ctx context.Context, typeID int64) error
	openNewMailWindow       func(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error
	openErr                 error
	openedFor               []int32
}

func (f *userInterfaceFake) open(ctx context.Context, characterID int32) (userInterfaceESI, error) {
	f.openedFor = append(f.openedFor, characterID)
	if f.openErr != nil {
		return nil, f.openErr
	}
	return f, nil
}

func (f *userInterfaceFake) SetAutopilotWaypoint(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
	if f.setAutopilotWaypoint == nil {
		return errors.New("userInterfaceFake: unexpected SetAutopilotWaypoint call")
	}
	return f.setAutopilotWaypoint(ctx, destinationID, addToBeginning, clearOtherWaypoints)
}

func (f *userInterfaceFake) OpenContractWindow(ctx context.Context, contractID int64) error {
	if f.openContractWindow == nil {
		return errors.New("userInterfaceFake: unexpected OpenContractWindow call")
	}
	return f.openContractWindow(ctx, contractID)
}

func (f *userInterfaceFake) OpenInformationWindow(ctx context.Context, targetID int64) error {
	if f.openInformationWindow == nil {
		return errors.New("userInterfaceFake: unexpected OpenInformationWindow call")
	}
	return f.openInformationWindow(ctx, targetID)
}

func (f *userInterfaceFake) OpenMarketDetailsWindow(ctx context.Context, typeID int64) error {
	if f.openMarketDetailsWindow == nil {
		return errors.New("userInterfaceFake: unexpected OpenMarketDetailsWindow call")
	}
	return f.openMarketDetailsWindow(ctx, typeID)
}

func (f *userInterfaceFake) OpenNewMailWindow(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
	if f.openNewMailWindow == nil {
		return errors.New("userInterfaceFake: unexpected OpenNewMailWindow call")
	}
	return f.openNewMailWindow(ctx, recipients, subject, body, toCorpOrAllianceID, toMailingListID)
}
