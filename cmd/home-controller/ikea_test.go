package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func ikeaTestDevice() Device {
	return Device{ID: "ikea_light", Name: "IKEA light", Type: "ikea", IKEA: &IKEATarget{Gateway: "test-gateway", Kind: "device", ID: 65537, Capabilities: IKEACapabilities{true, true, true, true}}}
}

// Explicitly authorized physical check; never runs in normal tests or CI.
func TestIKEARouterControlAndRestore(t *testing.T) {
	if os.Getenv("HC_IKEA_WRITE_TEST") != "1" {
		t.Skip("requires explicit physical-test authorization")
	}
	id, err := strconv.Atoi(os.Getenv("HC_IKEA_WRITE_TARGET"))
	if err != nil || id < 1 {
		t.Fatal("choose the authorized bulb ID")
	}
	var c ikeaCredentials
	if err = readJSON(base+"/ikea/credentials.json", &c); err != nil {
		t.Fatal(err)
	}
	var saved ikeaSession
	readJSON(base+"/state/ikea-auth.json", &saved)
	if saved.Key != "" {
		c.Identity, c.Key = saved.Identity, saved.Key
	}
	conn, err := ikeaDial(c.Address, c.Identity, c.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.conn.Close()
	path := fmt.Sprintf("/15001/%d", id)
	read := func() map[string]json.RawMessage {
		var raw map[string]json.RawMessage
		if err := conn.request(1, path, nil, &raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	original := read()
	target := ikeaDevice(original, c.Gateway, id)
	if !target.Reachable || !target.Capabilities.Color || !target.Capabilities.Temperature {
		t.Fatal("authorized colour bulb unavailable")
	}
	var controls []map[string]json.RawMessage
	json.Unmarshal(original["3311"], &controls)
	before := controls[0]
	backup := fmt.Sprintf("/tmp/home-controller-ikea-restore-%d.json", id)
	if exists(backup) {
		t.Fatal("previous restoration backup exists; resolve it first")
	}
	if err = atomicJSON(backup, original); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := restoreIKEALight(conn, path, before); err != nil {
			t.Error(err)
			return
		}
		os.Remove(backup)
		t.Log("Original power, brightness, hue, saturation and white temperature restored and verified")
	}()

	manager := &ikeaManager{}
	for _, test := range []struct {
		action string
		value  ActionValue
	}{{"brightness", ActionValue{Value: intPointer(30)}}, {"color", ActionValue{Color: "#00ff00"}}, {"hue", ActionValue{Value: intPointer(240)}}, {"saturation", ActionValue{Value: intPointer(25)}}, {"temperature", ActionValue{Value: intPointer(2700)}}, {"hue", ActionValue{Value: intPointer(190)}}, {"saturation", ActionValue{Value: intPointer(0)}}, {"hue", ActionValue{Value: intPointer(280)}}, {"saturation", ActionValue{Value: intPointer(60)}}} {
		target = ikeaDevice(read(), c.Gateway, id)
		if test.action == "hue" || test.action == "saturation" {
			target = manager.colourInput(target)
		}
		body, e := ikeaPayload(target, test.action, test.value)
		if e != nil {
			t.Fatal(e)
		}
		if e = conn.request(3, path, body, nil); e != nil {
			t.Fatal(e)
		}
		manager.recordMode(target, test.action, test.value)
		time.Sleep(3 * time.Second)
		after := ikeaDevice(read(), c.Gateway, id)
		switch test.action {
		case "brightness":
			if after.Brightness == nil || *after.Brightness != 30 {
				t.Fatal("brightness feedback differs", after.Brightness)
			}
		case "color":
			if after.Color != "#00ff00" {
				t.Fatal("colour feedback differs", after.Color)
			}
		case "hue":
			if after.Hue == nil || after.Saturation == nil || (*after.Saturation > 0 && *after.Hue != *test.value.Value) || *after.Saturation != *target.Saturation {
				t.Fatalf("hue/saturation feedback differs: colour=%s", after.Color)
			}
		case "saturation":
			if after.Saturation == nil || *after.Saturation != *test.value.Value || after.Hue == nil || (*after.Saturation > 0 && *after.Hue != *target.Hue) || after.Brightness == nil || *after.Brightness != 30 {
				t.Fatal("saturation changed hue or brightness")
			}
		case "temperature":
			if after.Temperature == nil || *after.Temperature != 2700 {
				t.Fatal("temperature feedback differs", after.Temperature)
			}
		}
		items := []IKEATargetInfo{after}
		manager.applyModes(items)
		t.Logf("%s confirmed: hue=%d saturation=%d brightness=%d white=%d", test.action, *after.Hue, *after.Saturation, *after.Brightness, *after.Temperature)
	}
}
func intPointer(v int) *int { return &v }

func TestIKEARouterRestoreOnly(t *testing.T) {
	if os.Getenv("HC_IKEA_RESTORE_TEST") != "1" {
		t.Skip("explicit restoration only")
	}
	id, err := strconv.Atoi(os.Getenv("HC_IKEA_WRITE_TARGET"))
	if err != nil || id < 1 {
		t.Fatal("choose the restoration target")
	}
	backup := fmt.Sprintf("/tmp/home-controller-ikea-restore-%d.json", id)
	var original map[string]json.RawMessage
	if err = readJSON(backup, &original); err != nil {
		t.Fatal(err)
	}
	var lights []map[string]json.RawMessage
	json.Unmarshal(original["3311"], &lights)
	if len(lights) != 1 {
		t.Fatal("invalid restoration record")
	}
	before := lights[0]
	var c ikeaCredentials
	if err = readJSON(base+"/ikea/credentials.json", &c); err != nil {
		t.Fatal(err)
	}
	var saved ikeaSession
	readJSON(base+"/state/ikea-auth.json", &saved)
	if saved.Key != "" {
		c.Identity, c.Key = saved.Identity, saved.Key
	}
	conn, err := ikeaDial(c.Address, c.Identity, c.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.conn.Close()
	path := fmt.Sprintf("/15001/%d", id)
	if err = restoreIKEALight(conn, path, before); err != nil {
		t.Fatal(err)
	}
	os.Remove(backup)
	t.Log("Original state restored and verified")
}

// The gateway rounds XY and H/S differently. Try both representations, always
// verifying the original readings before removing the recovery record.
func restoreIKEALight(conn *ikeaTransport, path string, before map[string]json.RawMessage) error {
	if err := conn.request(3, path, map[string]any{"3311": []any{map[string]any{"5711": before["5711"]}}}, nil); err != nil {
		return fmt.Errorf("white restoration failed; backup retained")
	}
	for _, keys := range [][]string{{"5709", "5710", "5851", "5850"}, {"5707", "5708", "5851", "5850"}} {
		body := map[string]any{}
		for _, k := range keys {
			body[k] = before[k]
		}
		if err := conn.request(3, path, map[string]any{"3311": []any{body}}, nil); err != nil {
			return fmt.Errorf("restoration failed; backup retained")
		}
		time.Sleep(2 * time.Second)
		var after map[string]json.RawMessage
		if err := conn.request(1, path, nil, &after); err != nil {
			return fmt.Errorf("could not read restoration; backup retained")
		}
		var light []map[string]json.RawMessage
		json.Unmarshal(after["3311"], &light)
		if len(light) != 1 {
			return fmt.Errorf("invalid restoration feedback; backup retained")
		}
		same := true
		for _, k := range []string{"5707", "5708", "5851", "5850", "5711", "5709", "5710"} {
			if rawInt(before, k) != rawInt(light[0], k) {
				same = false
			}
		}
		if same {
			return nil
		}
	}
	return fmt.Errorf("restoration feedback differs; backup retained")
}

func TestIKEAValidationAndScheduling(t *testing.T) {
	c := exampleConfig()
	c.Devices = append(c.Devices, ikeaTestDevice())
	if err := validate(c); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(radioConfig(c, nil), "ikea_light") {
		t.Fatal("IKEA device entered TellStick configuration")
	}
	for _, tc := range []struct {
		action string
		value  ActionValue
		valid  bool
	}{
		{"on", ActionValue{}, true}, {"brightness", ActionValue{Value: intPointer(0)}, true}, {"brightness", ActionValue{Value: intPointer(100)}, true},
		{"temperature", ActionValue{Value: intPointer(2700)}, true}, {"color", ActionValue{Color: "#ff8000"}, true},
		{"brightness", ActionValue{}, false}, {"brightness", ActionValue{Value: intPointer(101)}, false},
		{"temperature", ActionValue{Value: intPointer(9000)}, false}, {"color", ActionValue{Color: "red"}, false}, {"off", ActionValue{Value: intPointer(1)}, false},
	} {
		c.Schedules = []Schedule{{ID: "ikea_schedule", Enabled: true, Target: "ikea_light", Action: tc.action, ActionValue: tc.value, Days: []int{1}, Event: "time", Time: "08:00"}}
		if (validate(c) == nil) != tc.valid {
			t.Fatal("schedule validation", tc)
		}
		if !tc.valid {
			continue
		}
		s := testServer()
		s.config = c
		now := time.Date(2026, 11, 9, 7, 0, 1, 0, time.UTC)
		s.due(now.Add(-2*time.Second), now, map[string]bool{})
		if len(s.queue) != 1 || s.queue[0].Color != tc.value.Color || (tc.value.Value != nil && (s.queue[0].Value == nil || *s.queue[0].Value != *tc.value.Value)) {
			t.Fatal("schedule lost values", tc, s.queue)
		}
	}
	s := testServer()
	s.config.Devices = append(s.config.Devices, ikeaTestDevice())
	s.enqueueValue("ikea_light", "brightness", "manual", ActionValue{Value: intPointer(25)})
	s.enqueueValue("ikea_light", "color", "manual", ActionValue{Color: "#00ff00"})
	s.enqueueValue("ikea_light", "brightness", "manual", ActionValue{Value: intPointer(75)})
	if len(s.queue) != 2 || s.queue[0].Color != "#00ff00" || *s.queue[1].Value != 75 {
		t.Fatal("independent IKEA settings coalesced", s.queue)
	}
}
func TestIKEAInventoryAndPayloads(t *testing.T) {
	resources := map[string]string{
		"/15001": "[1,2,3,4]", "/15004": "[10]",
		"/15001/1":  `{"9001":"Colour lamp","5750":2,"9019":1,"3":{"1":"CWS"},"3311":[{"5850":1,"5851":127,"5711":370,"5707":0,"5708":65279}]}`,
		"/15001/2":  `{"9001":"Plug","5750":3,"9019":1,"3312":[{"5850":0,"5851":254}]}`,
		"/15001/3":  `{"9001":"Remote","5750":0,"9019":1}`,
		"/15001/4":  `{"9001":"Missing lamp","5750":2,"9019":0,"3311":[{"5850":1,"5851":254}]}`,
		"/15004/10": `{"9001":"Room","5850":0,"9018":{"15002":{"9003":[1,2,3,4,1]}}}`,
	}
	items, err := readIKEAInventory(func(path string, v any) error { return json.Unmarshal([]byte(resources[path]), v) }, "gateway")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatal("remote was exposed as a controllable target", items)
	}
	var bulb, plug, group IKEATargetInfo
	for _, i := range items {
		switch i.ID {
		case 1:
			bulb = i
		case 2:
			plug = i
		case 10:
			group = i
		}
	}
	if plug.Capabilities.Brightness || !bulb.Capabilities.Color || *bulb.Brightness != 50 || bulb.Color != "#ff0000" {
		t.Fatal("capabilities/state", items)
	}
	if !group.Partial || !group.Reachable || group.Power != nil || len(group.Members) != 3 || !group.Capabilities.Color {
		t.Fatal("mixed group", group)
	}
	if _, err := ikeaPayload(plug, "brightness", ActionValue{Value: intPointer(50)}); err == nil {
		t.Fatal("brightness sent to plug")
	}
	p, err := ikeaPayload(bulb, "color", ActionValue{Color: "#00ff00"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	if !strings.Contains(string(b), `"5707":21845`) || !strings.Contains(string(b), `"5850":1`) {
		t.Fatal(string(b))
	}
	p, _ = ikeaPayload(bulb, "brightness", ActionValue{Value: intPointer(0)})
	b, _ = json.Marshal(p)
	if !strings.Contains(string(b), `"5850":0`) {
		t.Fatal("zero brightness should switch off")
	}
	s := testServer()
	s.ikea = &ikeaManager{status: IKEAStatus{Connected: false, Targets: items}}
	d := ikeaTestDevice()
	d.IKEA = &bulb.IKEATarget
	if s.ikeaTarget(d).Reachable {
		t.Fatal("stale state reported reachable after gateway loss")
	}
}
func TestIKEAAPIAuthenticationAndValues(t *testing.T) {
	s := testServer()
	s.config.Devices = append(s.config.Devices, ikeaTestDevice())
	tok := strings.Repeat("b", 64)
	s.sessions = []Session{{Hash: hash(tok), CSRF: "csrf", Expires: time.Now().Add(time.Hour).Unix(), Revision: "revision"}}
	headers := map[string]string{"Cookie": cookieName + "=" + tok, "Content-Type": "application/json", "Authorization": "Bearer csrf"}
	body := `{"target":"ikea_light","action":"color","color":"#abcdef"}`
	if request(s, "POST", "/api/command", body, map[string]string{"Content-Type": "application/json"}).Code != 401 {
		t.Fatal("unauthenticated IKEA command")
	}
	if request(s, "POST", "/api/command", body, headers).Code != 202 || s.queue[0].Color != "#abcdef" {
		t.Fatal("colour command not retained")
	}
	for _, body := range []string{`{"target":"ikea_light","action":"brightness"}`, `{"target":"ikea_light","action":"color","color":"#123456","address":"127.0.0.1"}`, `{"target":"blind","action":"color","color":"#123456"}`} {
		if request(s, "POST", "/api/command", body, headers).Code != 400 {
			t.Fatal("invalid command accepted", body)
		}
	}
	delete(headers, "Authorization")
	if request(s, "POST", "/api/command", body, headers).Code != 403 {
		t.Fatal("IKEA CSRF bypass")
	}
}

// Explicit opt-in diagnostic for the actual ARM router. Reads only; uses existing
// credentials in place, never serializes them. Ordinary CI cannot reach a gateway.
func TestIKEARouterRead(t *testing.T) {
	if os.Getenv("HC_IKEA_READ_TEST") != "1" {
		t.Skip("router-only read check")
	}
	var c ikeaCredentials
	if err := readJSON(base+"/ikea/credentials.json", &c); err != nil {
		t.Fatal(err)
	}
	var saved ikeaSession
	readJSON(base+"/state/ikea-auth.json", &saved)
	if saved.Key != "" {
		c.Identity, c.Key = saved.Identity, saved.Key
	}
	conn, err := ikeaDial(c.Address, c.Identity, c.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.conn.Close()
	items, err := readIKEAInventory(func(path string, out any) error { return conn.request(1, path, nil, out) }, c.Gateway)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		t.Logf("%s %s: reachable=%v, capabilities=%+v", item.Kind, item.Name, item.Reachable, item.Capabilities)
	}
	if len(items) == 0 {
		t.Fatal("no targets")
	}
	if path := os.Getenv("HC_IKEA_INVENTORY"); path != "" {
		if filepath.Dir(path) != "/tmp" {
			t.Fatal("diagnostic output must be in /tmp")
		}
		if err := atomicJSON(path, IKEAStatus{Configured: true, Connected: true, Address: c.Address, Gateway: c.Gateway, Targets: items}); err != nil {
			t.Fatal(err)
		}
	}
}
