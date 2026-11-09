package main

import (
	"strconv"
	"time"
)

// The gateway does not expose a reliable white/colour mode flag. Remember only
// acknowledged changes, and invalidate that knowledge if colour state changes
// outside this controller. Unknown mode never displays a stale Kelvin value.
type ikeaMode struct {
	Mode      string `json:"mode"`
	At        int64  `json:"at"`
	Signature string `json:"signature"`
}

func ikeaModeKey(t IKEATargetInfo) string { return t.Gateway + ":" + strconv.Itoa(t.ID) }
func (m *ikeaManager) loadModes() {
	if m.modes == nil {
		m.modes = map[string]ikeaMode{}
		if m.modesPath != "" {
			readJSON(m.modesPath, &m.modes)
		}
		if m.modes == nil {
			m.modes = map[string]ikeaMode{}
		}
	}
}
func (m *ikeaManager) recordMode(t IKEATargetInfo, action string) {
	if !t.Capabilities.Color {
		return
	}
	mode := ""
	switch action {
	case "temperature":
		mode = "white"
	case "color", "hue", "saturation":
		mode = "color"
	}
	if mode == "" {
		return
	}
	m.loadModes()
	m.modes[ikeaModeKey(t)] = ikeaMode{Mode: mode, At: time.Now().Unix()}
	if m.modesPath != "" {
		atomicJSON(m.modesPath, m.modes)
	}
}
func aggregateIKEAValue(current **int, value *int) {
	if value == nil {
		return
	}
	v := *value
	if *current != nil && **current != v {
		v = -1
	}
	*current = &v
}
func (m *ikeaManager) applyModes(targets []IKEATargetInfo) {
	m.loadModes()
	seen := map[string]bool{}
	devices := map[int]IKEATargetInfo{}
	changed := false
	for i := range targets {
		t := &targets[i]
		if t.Kind != "device" {
			continue
		}
		key := ikeaModeKey(*t)
		seen[key] = true
		if t.Capabilities.Color && t.Reachable {
			if record, ok := m.modes[key]; ok {
				if time.Now().Unix()-record.At < 10 || record.Signature == "" {
					if record.Signature != t.colorSignature {
						record.Signature = t.colorSignature
						m.modes[key] = record
						changed = true
					}
					t.Mode = record.Mode
				} else if record.Signature == t.colorSignature {
					t.Mode = record.Mode
				} else {
					delete(m.modes, key)
					changed = true
				}
			}
		}
		devices[t.ID] = *t
	}
	for key := range m.modes {
		if !seen[key] {
			delete(m.modes, key)
			changed = true
		}
	}
	for i := range targets {
		t := &targets[i]
		if t.Kind != "group" {
			continue
		}
		first := true
		for _, id := range t.Members {
			d := devices[id]
			if !d.Reachable {
				continue
			}
			aggregateIKEAValue(&t.Hue, d.Hue)
			aggregateIKEAValue(&t.Saturation, d.Saturation)
			if d.Capabilities.Temperature || d.Capabilities.Color {
				if first {
					t.Mode = d.Mode
					first = false
				} else if t.Mode != d.Mode {
					t.Mode = "mixed"
				}
			}
		}
	}
	if changed && m.modesPath != "" {
		atomicJSON(m.modesPath, m.modes)
	}
}
