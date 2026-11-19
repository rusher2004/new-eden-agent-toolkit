package sso_test

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func TestScopesFor_AlwaysIncludesDefaultGroup(t *testing.T) {
	got, err := sso.ScopesFor([]string{"wallet"})
	if err != nil {
		t.Fatalf("ScopesFor: %v", err)
	}
	for _, want := range []string{"publicData", "esi-skills.read_skills.v1", "esi-wallet.read_character_wallet.v1"} {
		if !contains(got, want) {
			t.Errorf("ScopesFor([wallet]) is missing %s", want)
		}
	}
	if contains(got, "esi-mail.read_mail.v1") {
		t.Error("ScopesFor([wallet]) requested a scope from an unrequested group")
	}
}

func TestScopesFor_NoGroups_IsDefaultGroupOnly(t *testing.T) {
	got, err := sso.ScopesFor(nil)
	if err != nil {
		t.Fatalf("ScopesFor: %v", err)
	}
	if len(got) != len(sso.ScopeGroups[0].Scopes) {
		t.Errorf("ScopesFor(nil) = %v, want only the %s group", got, sso.DefaultScopeGroup)
	}
}

func TestScopesFor_UnknownGroup_Fails(t *testing.T) {
	_, err := sso.ScopesFor([]string{"walet"})
	if err == nil || !strings.Contains(err.Error(), "wallet") {
		t.Errorf("err = %v, want one listing the valid groups", err)
	}
}

func TestScopeGroups_NoScopeInTwoGroups(t *testing.T) {
	seen := map[string]string{}
	for _, g := range sso.ScopeGroups {
		for _, s := range g.Scopes {
			if other, ok := seen[s]; ok {
				t.Errorf("%s is in both %q and %q", s, other, g.Name)
			}
			seen[s] = g.Name
		}
	}
}

func TestGrantedGroups_OnlyFullyGrantedGroups(t *testing.T) {
	scopes, err := sso.ScopesFor([]string{"mail"})
	if err != nil {
		t.Fatalf("ScopesFor: %v", err)
	}
	scopes = append(scopes, "esi-fittings.read_fittings.v1") // half of "fittings"
	if got, want := sso.GrantedGroups(scopes), []string{"basic", "mail"}; !reflect.DeepEqual(got, want) {
		t.Errorf("GrantedGroups = %v, want %v", got, want)
	}
}

func TestScopesFromAccessToken(t *testing.T) {
	enc := base64.RawURLEncoding.EncodeToString
	tests := []struct {
		name    string
		payload string
		want    []string
	}{
		{"array", `{"scp":["publicData","esi-skills.read_skills.v1"]}`, []string{"publicData", "esi-skills.read_skills.v1"}},
		{"single string", `{"scp":"publicData"}`, []string{"publicData"}},
		{"none", `{}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sso.ExportedScopesFromAccessToken(enc([]byte(`{"alg":"RS256"}`)) + "." + enc([]byte(tt.payload)) + ".sig")
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("scopes = %v, want %v", got, tt.want)
			}
		})
	}
	if _, err := sso.ExportedScopesFromAccessToken("not-a-jwt"); err == nil {
		t.Error("non-JWT access token parsed without error")
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
