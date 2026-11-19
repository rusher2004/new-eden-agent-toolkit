package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestCollect_ListsEveryOperationSorted(t *testing.T) {
	s := spec{Paths: map[string]map[string]operation{
		"/b/{id}": {"get": {Security: []map[string][]string{{"OAuth2": {"scope-b"}}}}},
		"/a":      {"post": {}, "get": {Security: []map[string][]string{{"OAuth2": {"scope-a"}}}}},
	}}
	got, err := collect(s)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	want := []endpoint{{"GET", "/a", "scope-a"}, {"POST", "/a", ""}, {"GET", "/b/{id}", "scope-b"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collect = %+v, want %+v", got, want)
	}
}

func TestCollect_MultipleScopes_Fails(t *testing.T) {
	s := spec{Paths: map[string]map[string]operation{
		"/a": {"get": {Security: []map[string][]string{{"OAuth2": {"one", "two"}}}}},
	}}
	if _, err := collect(s); err == nil || !strings.Contains(err.Error(), "supports one") {
		t.Errorf("err = %v, want a multiple-scopes error", err)
	}
}

func TestCollect_AlternativeSecurityRequirements_Fails(t *testing.T) {
	s := spec{Paths: map[string]map[string]operation{
		"/a": {"get": {Security: []map[string][]string{{}, {"OAuth2": {"one"}}}}},
	}}
	if _, err := collect(s); err == nil || !strings.Contains(err.Error(), "alternative security requirements") {
		t.Errorf("err = %v, want an alternative-requirements error", err)
	}
}
