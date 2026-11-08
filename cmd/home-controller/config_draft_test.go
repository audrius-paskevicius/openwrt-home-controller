package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDraftDoesNotChangeActiveConfiguration(t *testing.T) {
	dir := t.TempDir()
	active := exampleConfig()
	if err := storeConfig(dir, active, true); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	draft := active
	draft.ControlOrder = []string{"group:all", "scheduler", "device:blind"}
	draft.SessionDays = 7
	if err := storeConfig(dir, draft, false); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if string(before) != string(after) {
		t.Fatal("Save changed active configuration")
	}
	got, pending, err := editableConfig(dir, active)
	if err != nil || !pending || !reflect.DeepEqual(got, draft) {
		t.Fatal("draft did not survive reload", got, pending, err)
	}
	if err := discardConfig(dir); err != nil {
		t.Fatal(err)
	}
	got, pending, err = editableConfig(dir, active)
	if err != nil || pending || !reflect.DeepEqual(got, active) {
		t.Fatal("Reset did not restore active settings")
	}
	if err := storeConfig(dir, draft, false); err != nil {
		t.Fatal(err)
	}
	if err := storeConfig(dir, draft, true); err != nil {
		t.Fatal(err)
	}
	var applied Config
	if err := readJSON(filepath.Join(dir, "config.json"), &applied); err != nil || !reflect.DeepEqual(applied, draft) {
		t.Fatal("Apply did not store settings", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pending.json")); !os.IsNotExist(err) {
		t.Fatal("Apply left draft behind")
	}
	invalid := draft
	invalid.ControlOrder = []string{"unknown"}
	if storeConfig(dir, invalid, false) == nil {
		t.Fatal("invalid draft accepted")
	}
	if _, err := os.Stat(filepath.Join(dir, "pending.json")); !os.IsNotExist(err) {
		t.Fatal("invalid draft written")
	}
}

func TestControlOrder(t *testing.T) {
	c := exampleConfig()
	if got := orderedControlKeys(c); !reflect.DeepEqual(got, []string{"device:blind", "scheduler", "group:all"}) {
		t.Fatal(got)
	}
	c.ControlOrder = []string{"group:all", "scheduler"}
	if validate(c) != nil {
		t.Fatal("partial order rejected")
	}
	if got := orderedControlKeys(c); !reflect.DeepEqual(got, []string{"group:all", "scheduler", "device:blind"}) {
		t.Fatal(got)
	}
	for _, order := range [][]string{{"scheduler", "scheduler"}, {"device:missing"}, {"all"}} {
		c.ControlOrder = order
		if validate(c) == nil {
			t.Fatal("invalid order accepted", order)
		}
	}
	c.ControlOrder = nil
	c.Devices[0].ID = "scheduler"
	c.Groups[0].Devices = []string{"scheduler"}
	if got := orderedControlKeys(c); !reflect.DeepEqual(got, []string{"device:scheduler", "scheduler", "group:all"}) {
		t.Fatal("typed IDs collided", got)
	}
}
