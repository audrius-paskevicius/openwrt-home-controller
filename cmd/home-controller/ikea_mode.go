package main

import (
	"math"
	"strconv"
	"time"
)

// The gateway does not expose a reliable white/colour mode flag. Remember only
// acknowledged changes, and invalidate that knowledge if colour state changes
// outside this controller. Unknown mode never displays a stale Kelvin value.
type ikeaMode struct {
	ikeaControls
	RawHue        int    `json:"raw_hue"`
	RawSaturation int    `json:"raw_saturation"`
	Mode          string `json:"mode"`
	At            int64  `json:"at"`
	Signature     string `json:"signature"`
}

// Slider selections are separate from measured state: white and colour are
// alternative modes of one lamp, not four independent physical properties.
type ikeaControls struct {
	Temperature *int `json:"temperature,omitempty"`
	Hue         *int `json:"hue,omitempty"`
	Saturation  *int `json:"saturation,omitempty"`
}

func (m *ikeaManager) controlRecord(t IKEATargetInfo) ikeaMode {
	m.loadModes()
	if r, ok := m.modes[ikeaModeKey(t)]; ok &&
		(time.Now().Unix()-r.At < 10 || r.Signature == t.colorSignature) &&
		r.Hue != nil && r.Saturation != nil {
		return r
	}
	return ikeaMode{ikeaControls: ikeaControls{Temperature: t.Temperature, Hue: t.Hue, Saturation: t.Saturation}, RawHue: t.rawHue, RawSaturation: t.rawSaturation}
}

func (m *ikeaManager) colourInput(t IKEATargetInfo) IKEATargetInfo {
	r := m.controlRecord(t)
	t.Hue, t.Saturation = r.Hue, r.Saturation
	t.rawHue, t.rawSaturation = r.RawHue, r.RawSaturation
	return t
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
func (m *ikeaManager) recordMode(t IKEATargetInfo, action string, v ActionValue) {
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
	r := m.controlRecord(t)
	r.Mode, r.At, r.Signature = mode, time.Now().Unix(), ""
	switch action {
	case "temperature":
		r.Temperature = v.Value
	case "hue":
		r.Hue = v.Value
		r.RawHue = int(math.Round(float64(*v.Value) / 360 * 65535))
	case "saturation":
		r.Saturation = v.Value
		r.RawSaturation = int(math.Round(float64(*v.Value) * 65279 / 100))
	case "color":
		h, s := ikeaHexHSV(v.Color)
		hue, sat := int(math.Round(h))%360, int(math.Round(s*100))
		r.Hue, r.Saturation = &hue, &sat
		r.RawHue, r.RawSaturation = int(math.Round(h/360*65535)), int(math.Round(s*65279))
	}
	m.modes[ikeaModeKey(t)] = r
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
				if t.Mode != "" && record.Hue != nil && record.Saturation != nil {
					controls := record.ikeaControls
					t.Controls = &controls
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
		controls := &ikeaControls{}
		for _, id := range t.Members {
			d := devices[id]
			if !d.Reachable {
				continue
			}
			aggregateIKEAValue(&t.Hue, d.Hue)
			aggregateIKEAValue(&t.Saturation, d.Saturation)
			c := d.Controls
			if c == nil {
				c = &ikeaControls{Temperature: d.Temperature, Hue: d.Hue, Saturation: d.Saturation}
			}
			aggregateIKEAValue(&controls.Temperature, c.Temperature)
			aggregateIKEAValue(&controls.Hue, c.Hue)
			aggregateIKEAValue(&controls.Saturation, c.Saturation)
			if d.Capabilities.Temperature || d.Capabilities.Color {
				if first {
					t.Mode = d.Mode
					first = false
				} else if t.Mode != d.Mode {
					t.Mode = "mixed"
				}
			}
		}
		t.Controls = controls
	}
	if changed && m.modesPath != "" {
		atomicJSON(m.modesPath, m.modes)
	}
}
