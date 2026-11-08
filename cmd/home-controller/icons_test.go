package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSymbolsValidationAndControlsState(t *testing.T) {
	for _, icon := range []string{"", "none", "blind", "clock", "lamp", "plug", "fan"} {
		c := exampleConfig()
		c.Devices[0].Icon = icon
		c.Groups[0].Icon = icon
		c.SchedulerIcon = icon
		raw, _ := json.Marshal(c)
		parsed, err := parseConfig(strings.NewReader(string(raw)))
		if err != nil || parsed.Devices[0].Icon != icon || parsed.Groups[0].Icon != icon || parsed.SchedulerIcon != icon {
			t.Fatal("symbol round trip failed", icon, err)
		}
	}
	for _, target := range []string{"device", "group", "scheduler"} {
		c := exampleConfig()
		switch target {
		case "device":
			c.Devices[0].Icon = "<svg>"
		case "group":
			c.Groups[0].Icon = "https://example.invalid/icon"
		default:
			c.SchedulerIcon = "unknown"
		}
		if validate(c) == nil {
			t.Fatal("invalid symbol accepted", target)
		}
	}
	s := testServer()
	s.config.Devices[0].Icon = "lamp"
	s.config.Groups[0].Icon = "none"
	s.config.SchedulerIcon = "fan"
	s.config.ControlOrder = []string{"group:all", "scheduler", "device:blind"}
	tok := strings.Repeat("a", 64)
	s.sessions = []Session{{Hash: hash(tok), CSRF: "csrf", Expires: time.Now().Add(time.Hour).Unix(), Revision: "revision"}}
	w := request(s, "GET", "/api/state", "", map[string]string{"Cookie": cookieName + "=" + tok})
	var state struct {
		Devices       []Device `json:"devices"`
		Groups        []Group  `json:"groups"`
		SchedulerIcon string   `json:"scheduler_icon"`
		ControlOrder  []string `json:"control_order"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &state) != nil || state.Devices[0].Icon != "lamp" || state.Groups[0].Icon != "none" || state.SchedulerIcon != "fan" {
		t.Fatal("symbols absent from control state", w.Code)
	}
	if strings.Join(state.ControlOrder, ",") != "group:all,scheduler,device:blind" {
		t.Fatal("control order absent from control state")
	}
}
