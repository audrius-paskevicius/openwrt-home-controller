package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func typedConfig() Config {
	c := exampleConfig()
	c.Devices = append(c.Devices, Device{ID: "lamp", Name: "Test switch", RadioID: 2, Type: "switch", Protocol: "arctech", HouseCode: "C", Unit: 7})
	c.Groups = append(c.Groups, Group{ID: "lights", Name: "Lights", Devices: []string{"lamp"}})
	return c
}

func TestDeviceTypesAndLegacyMigration(t *testing.T) {
	c := typedConfig()
	if err := validate(c); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(c)
	got, err := parseConfig(bytes.NewReader(b))
	if err != nil || got.Devices[0].deviceType() != "blind" || got.Devices[1].deviceType() != "switch" {
		t.Fatal(got, err)
	}
	for _, tc := range []struct {
		target, action string
		allowed        bool
	}{
		{"blind", "up", true}, {"all", "stop", true}, {"lamp", "on", true}, {"lights", "off", true},
		{"blind", "on", false}, {"lamp", "up", false}, {"lights", "stop", false}, {"missing", "off", false},
	} {
		if c.supportsAction(tc.target, tc.action) != tc.allowed {
			t.Fatal(tc)
		}
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.Devices[1].Type = "dimmer" },
		func(c *Config) { c.Devices[1].Protocol = "hasta" },
		func(c *Config) { c.Devices[1].HouseCode = "Q" },
		func(c *Config) { c.Devices[1].Unit = 0 },
		func(c *Config) { c.Devices[1].Unit = 17 },
		func(c *Config) { c.Groups[0].Devices = []string{"blind", "lamp"} },
		func(c *Config) { c.Groups[0].Devices = []string{"blind", "blind"} },
	} {
		bad := typedConfig()
		change(&bad)
		if validate(bad) == nil {
			t.Fatal("accepted invalid type/address/group", bad)
		}
	}
	c.Devices[1].HouseCode = "P"
	c.Devices[1].Unit = 16
	if err := validate(c); err != nil {
		t.Fatal("valid upper address boundary rejected", err)
	}
	text := radioConfig(c, map[string]string{"serial": "TEST_STICK"})
	for _, part := range []string{`protocol = "hasta"`, `model = "selflearningv2"`, `protocol = "arctech"`, `model = "codeswitch"`, `house = "P"`, `unit = 16`} {
		if !strings.Contains(text, part) {
			t.Fatal(part)
		}
	}
}

func TestSwitchCommandsAndSchedules(t *testing.T) {
	s := testServer()
	s.config = typedConfig()
	if s.enqueue("lamp", "up", "manual") == nil || len(s.queue) != 0 {
		t.Fatal("invalid action entered queue")
	}
	if err := s.enqueue("lights", "on", "manual"); err != nil {
		t.Fatal(err)
	}
	if len(s.queue) != 1 || s.queue[0].Device != "lamp" || s.queue[0].Action != "on" {
		t.Fatal(s.queue)
	}
	s.finishCommand(s.queue[0], "Sent (position unknown)", time.Now())
	if s.groupStates()["lights"].Sent != 1 {
		t.Fatal("switch group result missing")
	}
	rule := Schedule{ID: "night", Target: "lights", Action: "off", Days: []int{1}, Event: "time", Time: "22:00"}
	s.config.Schedules = []Schedule{rule}
	if err := validate(s.config); err != nil {
		t.Fatal(err)
	}
	s.config.Schedules[0].Action = "down"
	if validate(s.config) == nil {
		t.Fatal("movement schedule accepted for switch")
	}
	s.config.Schedules[0].Action = "off"
	s.config.Schedules[0].Target = "blind"
	if validate(s.config) == nil {
		t.Fatal("switch schedule accepted for blind")
	}
}
