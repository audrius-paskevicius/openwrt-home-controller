package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestIKEAIndependentSelectionsAndDelayedFeedback(t *testing.T) {
	d := ikeaTestDevice()
	initial := IKEATargetInfo{IKEATarget: *d.IKEA, Reachable: true, colorSignature: "colour", Hue: intPointer(240), Saturation: intPointer(60), Temperature: intPointer(2700), rawHue: 43690, rawSaturation: 39167}
	m := &ikeaManager{modesPath: filepath.Join(t.TempDir(), "modes.json")}
	m.recordMode(initial, "temperature", ActionValue{Value: intPointer(3000)})
	white := initial
	white.Hue, white.Saturation, white.rawHue, white.rawSaturation, white.colorSignature = intPointer(35), intPointer(12), 6371, 7833, "white"
	items := []IKEATargetInfo{white}
	m.applyModes(items)
	if *items[0].Controls.Hue != 240 || *items[0].Controls.Saturation != 60 || *items[0].Controls.Temperature != 3000 || *items[0].Hue != 35 {
		t.Fatal("white changed saved colour, or measured state was overwritten", items[0])
	}
	input := m.colourInput(white)
	if input.rawSaturation != 39167 {
		t.Fatal("hue command would use white-derived saturation")
	}
	m.recordMode(input, "hue", ActionValue{Value: intPointer(190)})
	// The gateway still reports the earlier white value when S is moved next.
	input = m.colourInput(white)
	if *input.Hue != 190 || input.rawHue != 34588 {
		t.Fatal("rapid saturation command would undo hue", input)
	}
	m.recordMode(input, "saturation", ActionValue{Value: intPointer(0)})
	grey := white
	grey.Hue, grey.Saturation, grey.rawHue, grey.rawSaturation, grey.colorSignature = intPointer(0), intPointer(0), 0, 0, "grey"
	items = []IKEATargetInfo{grey}
	m.applyModes(items)
	// Saturation zero has no measured hue; retain the user's hue for later.
	if *items[0].Controls.Hue != 190 || *items[0].Controls.Saturation != 0 || *items[0].Controls.Temperature != 3000 {
		t.Fatal("zero saturation discarded an independent setting")
	}
	reloaded := &ikeaManager{modesPath: m.modesPath}
	input = reloaded.colourInput(grey)
	if *input.Hue != 190 || input.rawHue != 34588 {
		t.Fatal("restart discarded selected hue")
	}
	key := ikeaModeKey(grey)
	r := reloaded.modes[key]
	r.At = time.Now().Add(-time.Minute).Unix()
	reloaded.modes[key] = r
	external := grey
	external.Hue, external.rawHue, external.colorSignature = intPointer(80), 14564, "IKEA app change"
	input = reloaded.colourInput(external)
	if input.rawHue != external.rawHue {
		t.Fatal("external app change ignored by next command")
	}
	items = []IKEATargetInfo{external}
	reloaded.applyModes(items)
	if items[0].Controls != nil || items[0].Mode != "" {
		t.Fatal("external app change hidden by old selections")
	}
}

func TestIKEASettlingKeepsCompleteGroupSnapshot(t *testing.T) {
	previous := IKEAStatus{Connected: true, Targets: []IKEATargetInfo{{IKEATarget: IKEATarget{Kind: "group", ID: 10}, Brightness: intPointer(40)}}}
	m := &ikeaManager{status: previous, settleUntil: time.Now().Add(3 * time.Second)}
	// Without the settle guard refresh would try missing credentials and publish
	// a disconnected state. No gateway reads may publish a partial transition.
	m.refresh()
	if got := m.snapshot(); !got.Connected || len(got.Targets) != 1 || *got.Targets[0].Brightness != 40 {
		t.Fatal("transition replaced the last complete snapshot")
	}
	if delay := m.refreshDelay(); delay <= 0 || delay > 3*time.Second {
		t.Fatal("did not schedule prompt settled readback", delay)
	}
	m.settleUntil = time.Now().Add(-time.Second)
	if m.refreshDelay() != 15*time.Second {
		t.Fatal("normal polling not restored")
	}
}
