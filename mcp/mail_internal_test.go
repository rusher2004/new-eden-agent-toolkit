package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func newTestStoreWithCharacter(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

func newTestStoreWithoutCharacter(t *testing.T) *config.Store {
	t.Helper()
	keyring.MockInit()
	return config.NewStoreAt(t.TempDir() + "/config.json")
}

// --- list_character_mail ---

func TestListCharacterMailHandler_MapsFields(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := listCharacterMail
	t.Cleanup(func() { listCharacterMail = orig })
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mailID1, mailID2, from := int64(200), int64(100), int64(90000001)
	subject := "Hi"
	isRead := true
	var gotLabelIDs []int64
	var gotLastMailID int64
	listCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, labelIDs []int64, lastMailID int64) ([]MailHeader, error) {
		gotLabelIDs = labelIDs
		gotLastMailID = lastMailID
		return []MailHeader{
			{MailID: &mailID1, From: &from, Subject: &subject, Timestamp: &ts, IsRead: &isRead, Labels: []int64{1, 2}, Recipients: []MailRecipient{{RecipientID: 90000002, RecipientType: "character"}}},
			{MailID: &mailID2, From: &from, Subject: &subject, Timestamp: &ts, IsRead: &isRead, Labels: []int64{1}, Recipients: nil},
		}, nil
	}

	handler := listCharacterMailHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterMailInput{CharacterID: 7, LabelIDs: []int64{1, 2}, LastMailID: 300})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if len(gotLabelIDs) != 2 || gotLabelIDs[0] != 1 || gotLabelIDs[1] != 2 {
		t.Errorf("captured labelIDs = %v, want [1 2]", gotLabelIDs)
	}
	if gotLastMailID != 300 {
		t.Errorf("captured lastMailID = %d, want 300", gotLastMailID)
	}

	if out.CharacterID != 7 {
		t.Errorf("out.CharacterID = %d, want 7", out.CharacterID)
	}
	if len(out.Mail) != 2 {
		t.Fatalf("out.Mail = %+v, want two entries", out.Mail)
	}
	m0 := out.Mail[0]
	if m0.MailID == nil || *m0.MailID != 200 || m0.From == nil || *m0.From != 90000001 || m0.Subject == nil || *m0.Subject != "Hi" {
		t.Errorf("Mail[0] = %+v, want MailID=200 From=90000001 Subject=\"Hi\"", m0)
	}
	if m0.Timestamp == nil || !m0.Timestamp.Equal(ts) {
		t.Errorf("Mail[0].Timestamp = %v, want %v", m0.Timestamp, ts)
	}
	if m0.IsRead == nil || !*m0.IsRead {
		t.Errorf("Mail[0].IsRead = %v, want true", m0.IsRead)
	}
	if len(m0.Labels) != 2 || m0.Labels[0] != 1 || m0.Labels[1] != 2 {
		t.Errorf("Mail[0].Labels = %v, want [1 2]", m0.Labels)
	}
	if len(m0.Recipients) != 1 || m0.Recipients[0].RecipientID != 90000002 || m0.Recipients[0].RecipientType != "character" {
		t.Errorf("Mail[0].Recipients = %+v, want one entry {RecipientID:90000002 RecipientType:\"character\"}", m0.Recipients)
	}

	// NextLastMailID must be the lowest MailID across the batch (100), so the caller can pass it
	// straight back as lastMailId to page further backwards.
	if out.NextLastMailID == nil || *out.NextLastMailID != 100 {
		t.Errorf("out.NextLastMailID = %v, want 100", out.NextLastMailID)
	}
}

func TestListCharacterMailHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := listCharacterMail
	t.Cleanup(func() { listCharacterMail = orig })
	listCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, labelIDs []int64, lastMailID int64) ([]MailHeader, error) {
		t.Fatal("listCharacterMail should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMailHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterMailInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- get_character_mail_labels ---

func TestGetCharacterMailLabelsHandler_MapsFields(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := getCharacterMailLabels
	t.Cleanup(func() { getCharacterMailLabels = orig })
	labelID, unread, total := int64(1), int64(3), int64(9)
	name, color := "Inbox", "#ffffff"
	getCharacterMailLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MailLabel, *int64, error) {
		return []MailLabel{{LabelID: &labelID, Name: &name, Color: &color, UnreadCount: &unread}}, &total, nil
	}

	handler := getCharacterMailLabelsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterMailLabelsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 {
		t.Errorf("out.CharacterID = %d, want 7", out.CharacterID)
	}
	if out.TotalUnreadCount == nil || *out.TotalUnreadCount != 9 {
		t.Errorf("out.TotalUnreadCount = %v, want 9", out.TotalUnreadCount)
	}
	if len(out.Labels) != 1 {
		t.Fatalf("out.Labels = %+v, want one entry", out.Labels)
	}
	l := out.Labels[0]
	if l.LabelID == nil || *l.LabelID != 1 || l.Name == nil || *l.Name != "Inbox" || l.Color == nil || *l.Color != "#ffffff" || l.UnreadCount == nil || *l.UnreadCount != 3 {
		t.Errorf("out.Labels[0] = %+v, want LabelID=1 Name=\"Inbox\" Color=\"#ffffff\" UnreadCount=3", l)
	}
}

func TestGetCharacterMailLabelsHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := getCharacterMailLabels
	t.Cleanup(func() { getCharacterMailLabels = orig })
	getCharacterMailLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MailLabel, *int64, error) {
		t.Fatal("getCharacterMailLabels should not be called for an unknown character")
		return nil, nil, nil
	}

	handler := getCharacterMailLabelsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterMailLabelsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_character_mail_lists ---

func TestListCharacterMailListsHandler_MapsFields(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := listCharacterMailLists
	t.Cleanup(func() { listCharacterMailLists = orig })
	listCharacterMailLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MailList, error) {
		return []MailList{{MailingListID: 1, Name: "Test List"}}, nil
	}

	handler := listCharacterMailListsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterMailListsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.MailLists) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one mail list", out)
	}
	if out.MailLists[0].MailingListID != 1 || out.MailLists[0].Name != "Test List" {
		t.Errorf("out.MailLists[0] = %+v, want MailingListID=1 Name=\"Test List\"", out.MailLists[0])
	}
}

func TestListCharacterMailListsHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := listCharacterMailLists
	t.Cleanup(func() { listCharacterMailLists = orig })
	listCharacterMailLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]MailList, error) {
		t.Fatal("listCharacterMailLists should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMailListsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterMailListsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- get_character_mail ---

func TestGetCharacterMailHandler_MapsFields(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := getCharacterMail
	t.Cleanup(func() { getCharacterMail = orig })
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	from := int64(90000001)
	subject, body := "Hi", "Hello there"
	isRead := true
	var gotMailID int64
	getCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, mailID int64) (GetCharacterMailOutput, error) {
		gotMailID = mailID
		// Deliberately leave CharacterID unset here -- the handler is responsible for filling it
		// in from the input, same as get_character_fatigue's pattern in mcp/character.go.
		return GetCharacterMailOutput{
			MailID: mailID, From: &from, Subject: &subject, Body: &body, Timestamp: &ts, Read: &isRead,
			Labels: []int64{1}, Recipients: []MailRecipient{{RecipientID: 90000002, RecipientType: "character"}},
		}, nil
	}

	handler := getCharacterMailHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetCharacterMailInput{CharacterID: 7, MailID: 123})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotMailID != 123 {
		t.Errorf("captured mailID = %d, want 123", gotMailID)
	}
	if out.CharacterID != 7 || out.MailID != 123 {
		t.Errorf("out.CharacterID/MailID = %d/%d, want 7/123", out.CharacterID, out.MailID)
	}
	if out.From == nil || *out.From != 90000001 || out.Subject == nil || *out.Subject != "Hi" || out.Body == nil || *out.Body != "Hello there" {
		t.Errorf("out = %+v, want From=90000001 Subject=\"Hi\" Body=\"Hello there\"", out)
	}
	if out.Timestamp == nil || !out.Timestamp.Equal(ts) {
		t.Errorf("out.Timestamp = %v, want %v", out.Timestamp, ts)
	}
	if out.Read == nil || !*out.Read {
		t.Errorf("out.Read = %v, want true", out.Read)
	}
	if len(out.Labels) != 1 || out.Labels[0] != 1 {
		t.Errorf("out.Labels = %v, want [1]", out.Labels)
	}
	if len(out.Recipients) != 1 || out.Recipients[0].RecipientID != 90000002 || out.Recipients[0].RecipientType != "character" {
		t.Errorf("out.Recipients = %+v, want one entry {RecipientID:90000002 RecipientType:\"character\"}", out.Recipients)
	}
}

func TestGetCharacterMailHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := getCharacterMail
	t.Cleanup(func() { getCharacterMail = orig })
	getCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, mailID int64) (GetCharacterMailOutput, error) {
		t.Fatal("getCharacterMail should not be called for an unknown character")
		return GetCharacterMailOutput{}, nil
	}

	handler := getCharacterMailHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetCharacterMailInput{CharacterID: 999, MailID: 123})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- send_character_mail ---

func TestSendCharacterMailHandler_MapsFields(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := sendCharacterMail
	t.Cleanup(func() { sendCharacterMail = orig })
	var gotSubject, gotBody string
	var gotRecipients []MailRecipientInput
	var gotApprovedCost *int64
	sendCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, subject, body string, recipients []MailRecipientInput, approvedCost *int64) (int64, error) {
		gotSubject, gotBody, gotRecipients, gotApprovedCost = subject, body, recipients, approvedCost
		return 789, nil
	}

	cost := int64(5000)
	input := SendCharacterMailInput{
		CharacterID: 7, Subject: "Hi", Body: "Hello there",
		Recipients:   []MailRecipientInput{{RecipientID: 90000002, RecipientType: "character"}},
		ApprovedCost: &cost,
	}
	handler := sendCharacterMailHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if gotSubject != "Hi" || gotBody != "Hello there" {
		t.Errorf("captured subject/body = %q/%q, want \"Hi\"/\"Hello there\"", gotSubject, gotBody)
	}
	if len(gotRecipients) != 1 || gotRecipients[0].RecipientID != 90000002 || gotRecipients[0].RecipientType != "character" {
		t.Errorf("captured recipients = %+v, want one entry {RecipientID:90000002 RecipientType:\"character\"}", gotRecipients)
	}
	if gotApprovedCost == nil || *gotApprovedCost != 5000 {
		t.Errorf("captured approvedCost = %v, want 5000", gotApprovedCost)
	}
	if out.CharacterID != 7 || out.MailID != 789 {
		t.Errorf("out = %+v, want CharacterID=7 MailID=789", out)
	}
}

func TestSendCharacterMailHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := sendCharacterMail
	t.Cleanup(func() { sendCharacterMail = orig })
	sendCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, subject, body string, recipients []MailRecipientInput, approvedCost *int64) (int64, error) {
		t.Fatal("sendCharacterMail should not be called for an unknown character")
		return 0, nil
	}

	handler := sendCharacterMailHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, SendCharacterMailInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- create_character_mail_label ---

func TestCreateCharacterMailLabelHandler_MapsFields(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := createCharacterMailLabel
	t.Cleanup(func() { createCharacterMailLabel = orig })
	var gotName string
	var gotColor *string
	createCharacterMailLabel = func(ctx context.Context, clientID string, characterID int32, userAgent string, name string, color *string) (int64, error) {
		gotName, gotColor = name, color
		return 42, nil
	}

	color := "#ffffff"
	handler := createCharacterMailLabelHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, CreateCharacterMailLabelInput{CharacterID: 7, Name: "Important", Color: &color})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotName != "Important" {
		t.Errorf("captured name = %q, want \"Important\"", gotName)
	}
	if gotColor == nil || *gotColor != "#ffffff" {
		t.Errorf("captured color = %v, want \"#ffffff\"", gotColor)
	}
	if out.CharacterID != 7 || out.LabelID != 42 {
		t.Errorf("out = %+v, want CharacterID=7 LabelID=42", out)
	}
}

func TestCreateCharacterMailLabelHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := createCharacterMailLabel
	t.Cleanup(func() { createCharacterMailLabel = orig })
	createCharacterMailLabel = func(ctx context.Context, clientID string, characterID int32, userAgent string, name string, color *string) (int64, error) {
		t.Fatal("createCharacterMailLabel should not be called for an unknown character")
		return 0, nil
	}

	handler := createCharacterMailLabelHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, CreateCharacterMailLabelInput{CharacterID: 999, Name: "Important"})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- update_character_mail ---

func TestUpdateCharacterMailHandler_Succeeds(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := updateCharacterMail
	t.Cleanup(func() { updateCharacterMail = orig })
	var gotMailID int64
	var gotLabels []int64
	var gotRead *bool
	updateCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, mailID int64, labels []int64, read *bool) error {
		gotMailID, gotLabels, gotRead = mailID, labels, read
		return nil
	}

	read := true
	handler := updateCharacterMailHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, UpdateCharacterMailInput{CharacterID: 7, MailID: 123, Labels: []int64{1, 2}, Read: &read})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotMailID != 123 {
		t.Errorf("captured mailID = %d, want 123", gotMailID)
	}
	if len(gotLabels) != 2 || gotLabels[0] != 1 || gotLabels[1] != 2 {
		t.Errorf("captured labels = %v, want [1 2]", gotLabels)
	}
	if gotRead == nil || !*gotRead {
		t.Errorf("captured read = %v, want true", gotRead)
	}
	if out.CharacterID != 7 || out.MailID != 123 {
		t.Errorf("out = %+v, want CharacterID=7 MailID=123", out)
	}
}

func TestUpdateCharacterMailHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := updateCharacterMail
	t.Cleanup(func() { updateCharacterMail = orig })
	updateCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, mailID int64, labels []int64, read *bool) error {
		t.Fatal("updateCharacterMail should not be called for an unknown character")
		return nil
	}

	handler := updateCharacterMailHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, UpdateCharacterMailInput{CharacterID: 999, MailID: 123})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- delete_character_mail ---

func TestDeleteCharacterMailHandler_Succeeds(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := deleteCharacterMail
	t.Cleanup(func() { deleteCharacterMail = orig })
	deleteCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, mailID int64) error {
		return nil
	}

	handler := deleteCharacterMailHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, DeleteCharacterMailInput{CharacterID: 7, MailID: 123})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.MailID != 123 {
		t.Errorf("out = %+v, want CharacterID=7 MailID=123", out)
	}
}

func TestDeleteCharacterMailHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := deleteCharacterMail
	t.Cleanup(func() { deleteCharacterMail = orig })
	deleteCharacterMail = func(ctx context.Context, clientID string, characterID int32, userAgent string, mailID int64) error {
		t.Fatal("deleteCharacterMail should not be called for an unknown character")
		return nil
	}

	handler := deleteCharacterMailHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, DeleteCharacterMailInput{CharacterID: 999, MailID: 123})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- delete_character_mail_label ---

func TestDeleteCharacterMailLabelHandler_Succeeds(t *testing.T) {
	store := newTestStoreWithCharacter(t)

	orig := deleteCharacterMailLabel
	t.Cleanup(func() { deleteCharacterMailLabel = orig })
	deleteCharacterMailLabel = func(ctx context.Context, clientID string, characterID int32, userAgent string, labelID int64) error {
		return nil
	}

	handler := deleteCharacterMailLabelHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, DeleteCharacterMailLabelInput{CharacterID: 7, LabelID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.LabelID != 1 {
		t.Errorf("out = %+v, want CharacterID=7 LabelID=1", out)
	}
}

func TestDeleteCharacterMailLabelHandler_UnknownCharacter(t *testing.T) {
	store := newTestStoreWithoutCharacter(t)

	orig := deleteCharacterMailLabel
	t.Cleanup(func() { deleteCharacterMailLabel = orig })
	deleteCharacterMailLabel = func(ctx context.Context, clientID string, characterID int32, userAgent string, labelID int64) error {
		t.Fatal("deleteCharacterMailLabel should not be called for an unknown character")
		return nil
	}

	handler := deleteCharacterMailLabelHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, DeleteCharacterMailLabelInput{CharacterID: 999, LabelID: 1})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
