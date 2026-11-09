package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPreferredConnectedController(t *testing.T) {
	a := map[string]string{"serial": "TEST_A", "bus": "1", "device": "2"}
	b := map[string]string{"serial": "TEST_B", "bus": "1", "device": "3"}
	for _, tc := range []struct {
		available                 []map[string]string
		preferred, previous, want string
	}{
		{[]map[string]string{a, b}, "TEST_B", "TEST_A", "TEST_B"},
		{[]map[string]string{a, b}, "", "TEST_B", "TEST_B"},
		{[]map[string]string{a}, "TEST_B", "TEST_B", "TEST_A"},
		{nil, "TEST_A", "TEST_A", ""},
	} {
		if got := preferredController(tc.available, tc.preferred, tc.previous)["serial"]; got != tc.want {
			t.Fatalf("got %q, want %q", got, tc.want)
		}
	}
	if !sameController(a, a) || sameController(a, b) || sameController(a, map[string]string{"serial": "TEST_A", "bus": "1", "device": "9"}) {
		t.Fatal("USB identity must distinguish reconnection from an unchanged spare")
	}
	if submittedPreference("TEST_A", "1:2", []map[string]string{a}) != "TEST_A" || submittedPreference("TEST_A", "1:9", []map[string]string{a}) != "" || submittedPreference("TEST_A", "", []map[string]string{a}) != "" {
		t.Fatal("stale form restored a reconnected device preference")
	}
}

func TestUnplugForgetsPreferenceAndPreservesDraft(t *testing.T) {
	dir := t.TempDir()
	a := exampleConfig()
	a.PreferredController = "TEST_A"
	draft := a
	draft.SessionDays = 7
	draft.PreferredController = "TEST_B"
	if err := storeConfig(dir, a, true); err != nil {
		t.Fatal(err)
	}
	if err := storeConfig(dir, draft, false); err != nil {
		t.Fatal(err)
	}
	available := []map[string]string{{"serial": "TEST_B"}}
	if err := clearMissingPreferences(dir, available); err != nil {
		t.Fatal(err)
	}
	var got Config
	if err := readJSON(filepath.Join(dir, "config.json"), &got); err != nil {
		t.Fatal(err)
	}
	a.PreferredController = ""
	if !reflect.DeepEqual(a, got) {
		t.Fatal("unplug modified unrelated active settings")
	}
	if err := readJSON(filepath.Join(dir, "pending.json"), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(draft, got) {
		t.Fatal("unplug modified an available draft preference or unrelated edits")
	}
	if err := clearMissingPreferences(dir, nil); err != nil {
		t.Fatal(err)
	}
	if err := clearMissingPreferences(dir, []map[string]string{{"serial": "TEST_A"}, {"serial": "TEST_B"}}); err != nil {
		t.Fatal(err)
	}
	got = Config{}
	if err := readJSON(filepath.Join(dir, "pending.json"), &got); err != nil {
		t.Fatal(err)
	}
	draft.PreferredController = ""
	if !reflect.DeepEqual(draft, got) {
		t.Fatal("reconnect restored a forgotten preference")
	}
	before, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if err := clearMissingPreferences(dir, nil); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if string(before) != string(after) {
		t.Fatal("unchanged inventory rewrote settings")
	}
}

func TestPreferredControllerValidation(t *testing.T) {
	for _, serial := range []string{"TEST_A", "", "TEST1234"} {
		c := exampleConfig()
		c.PreferredController = serial
		if err := validate(c); err != nil {
			t.Fatal(err)
		}
	}
	for _, serial := range []string{"../other", "bad\nserial", "\";command", string(make([]byte, 65))} {
		c := exampleConfig()
		c.PreferredController = serial
		if validate(c) == nil {
			t.Fatal("accepted invalid serial")
		}
	}
}
