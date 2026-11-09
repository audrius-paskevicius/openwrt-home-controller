package main

import (
	"strings"
	"testing"
	"time"
)

func dimmerConfig() Config {
	c := typedConfig()
	c.Devices = append(c.Devices, Device{ID: "dimmer", Name: "Dimmer", RadioID: 3, Type: "dimmer", Protocol: "arctech", HouseCode: "D", Unit: 4})
	c.Groups = append(c.Groups, Group{ID: "dimmers", Name: "Dimmers", Devices: []string{"dimmer"}})
	return c
}

func TestDimmerProfileAndCommands(t *testing.T) {
	c := dimmerConfig()
	if err := validate(c); err != nil {
		t.Fatal(err)
	}
	d := c.Devices[2]
	for _, tc := range []struct{ action, wire string }{{"toggle", "off"}, {"dim", "on"}} {
		if !validAction(tc.action) || !c.supportsAction("dimmer", tc.action) || !c.supportsAction("dimmers", tc.action) || d.radioAction(tc.action) != tc.wire {
			t.Fatal(tc)
		}
	}
	for _, action := range []string{"on", "off", "up", "stop", "down"} {
		if c.supportsAction("dimmer", action) {
			t.Fatal("absolute or movement action accepted", action)
		}
	}
	if c.supportsAction("lamp", "toggle") || c.supportsAction("blind", "dim") || c.Devices[1].radioAction("on") != "on" {
		t.Fatal("dimmer mapping leaked into other device types")
	}
	if !strings.Contains(radioConfig(c, map[string]string{"serial": "TEST"}), "house = \"D\"\n unit = 4") {
		t.Fatal("dimmer address not generated")
	}
	for _, target := range []string{"dimmer", "dimmers"} {
		c.Schedules = []Schedule{{ID: "night", Target: target, Action: "toggle", Days: []int{1}, Event: "time", Time: "22:00"}}
		if err := validate(c); err != nil {
			t.Fatal("dimmer toggle schedule rejected", target, err)
		}
		c.Schedules[0].Action = "dim"
		if err := validate(c); err != nil {
			t.Fatal("dimmer cycle schedule rejected", target, err)
		}
		c.Schedules[0].Action = "off"
		if validate(c) == nil {
			t.Fatal("absolute Off schedule accepted for toggle dimmer", target)
		}
	}
}

func TestDimmerQueuePreservesPressesAndBoundsGroups(t *testing.T) {
	s := testServer()
	s.config = dimmerConfig()
	for _, action := range []string{"toggle", "dim", "dim", "toggle"} {
		if err := s.enqueue("dimmer", action, "manual"); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.queue) != 4 {
		t.Fatal("toggle presses were coalesced", s.queue)
	}
	for i, want := range []string{"toggle", "dim", "dim", "toggle"} {
		if s.queue[i].Action != want {
			t.Fatal("press order changed", s.queue)
		}
	}
	other := s.config.Devices[2]
	other.ID, other.RadioID, other.Unit = "other", 4, 5
	s.config.Devices = append(s.config.Devices, other)
	s.config.Groups[2].Devices = []string{"other", "dimmer"}
	request := s.nextRequest
	if s.enqueue("dimmers", "toggle", "manual") == nil || len(s.queue) != 4 || s.nextRequest != request {
		t.Fatal("full dimmer queue did not reject whole group atomically")
	}
	if err := s.enqueue("other", "toggle", "schedule"); err != nil {
		t.Fatal("scheduled dimmer command rejected", err)
	}
}

func TestDimmerAPIValidationAndPrivacy(t *testing.T) {
	s := testServer()
	s.config = dimmerConfig()
	tok := strings.Repeat("b", 64)
	s.sessions = []Session{{Hash: hash(tok), CSRF: "csrf", Expires: time.Now().Add(time.Hour).Unix(), Revision: "revision"}}
	headers := map[string]string{"Cookie": cookieName + "=" + tok, "Content-Type": "application/json", "Authorization": "Bearer csrf"}
	for _, action := range []string{"toggle", "dim"} {
		body := `{"target":"dimmer","action":"` + action + `"}`
		if request(s, "POST", "/api/command", body, map[string]string{"Content-Type": "application/json"}).Code != 401 || request(s, "POST", "/api/command", body, headers).Code != 202 {
			t.Fatal("dimmer authentication/action validation failed", action)
		}
	}
	if request(s, "POST", "/api/command", `{"target":"dimmer","action":"off"}`, headers).Code != 400 {
		t.Fatal("raw Off bypassed dimmer semantics")
	}
	state := request(s, "GET", "/api/state", "", headers).Body.String()
	if !strings.Contains(state, `"type":"dimmer"`) || strings.Contains(state, `"house_code"`) || strings.Contains(state, `"radio_id"`) {
		t.Fatal("incorrect dimmer state projection", state)
	}
}
