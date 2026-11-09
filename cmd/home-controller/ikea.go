package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/codes"
)

type ActionValue struct {
	Value *int   `json:"value,omitempty"`
	Color string `json:"color,omitempty"`
}
type IKEACapabilities struct {
	Power       bool `json:"power"`
	Brightness  bool `json:"brightness"`
	Temperature bool `json:"temperature"`
	Color       bool `json:"color"`
}

func (c IKEACapabilities) supports(action string) bool {
	switch action {
	case "on", "off":
		return c.Power
	case "brightness":
		return c.Brightness
	case "temperature":
		return c.Temperature
	case "color":
		return c.Color
	}
	return false
}

type IKEATarget struct {
	Gateway      string           `json:"gateway"`
	Kind         string           `json:"kind"`
	ID           int              `json:"id"`
	Capabilities IKEACapabilities `json:"capabilities"`
}
type IKEATargetInfo struct {
	IKEATarget
	Name        string `json:"name"`
	Model       string `json:"model,omitempty"`
	Reachable   bool   `json:"reachable"`
	Partial     bool   `json:"partial,omitempty"`
	Power       *bool  `json:"power,omitempty"`
	Brightness  *int   `json:"brightness,omitempty"`
	Temperature *int   `json:"temperature,omitempty"`
	Color       string `json:"color,omitempty"`
	Members     []int  `json:"members,omitempty"`
}
type IKEAStatus struct {
	Configured bool             `json:"configured"`
	Connected  bool             `json:"connected"`
	Address    string           `json:"address,omitempty"`
	Gateway    string           `json:"gateway,omitempty"`
	Error      string           `json:"error,omitempty"`
	Targets    []IKEATargetInfo `json:"targets"`
}
type ikeaCredentials struct {
	Address  string `json:"address"`
	Gateway  string `json:"gateway_id"`
	Identity string `json:"identity,omitempty"`
	Key      string `json:"key,omitempty"`
	Code     string `json:"security_code"`
}
type ikeaSession struct {
	Fingerprint string `json:"fingerprint"`
	Identity    string `json:"identity"`
	Key         string `json:"key"`
}
type ikeaManager struct {
	network                                  sync.Mutex
	mu                                       sync.RWMutex
	status                                   IKEAStatus
	conn                                     *ikeaTransport
	fingerprint                              string
	credentialsPath, sessionPath, statusPath string
	wake                                     chan bool
}

func newIKEA() *ikeaManager {
	return &ikeaManager{credentialsPath: base + "/ikea/credentials.json", sessionPath: base + "/state/ikea-auth.json", statusPath: run + "/ikea.json", wake: make(chan bool, 1)}
}
func (m *ikeaManager) snapshot() IKEAStatus { m.mu.RLock(); defer m.mu.RUnlock(); return m.status }
func (m *ikeaManager) publish(v IKEAStatus) {
	m.mu.Lock()
	m.status = v
	m.mu.Unlock()
	atomicJSON(m.statusPath, v)
}
func (m *ikeaManager) close() {
	if m.conn != nil {
		m.conn.conn.Close()
		m.conn = nil
	}
}
func (m *ikeaManager) connect(c ikeaCredentials, fingerprint string) error {
	var saved ikeaSession
	readJSON(m.sessionPath, &saved)
	identity, key := c.Identity, c.Key
	if saved.Fingerprint == fingerprint {
		identity, key = saved.Identity, saved.Key
	}
	if identity != "" && key != "" {
		t, err := ikeaDial(c.Address, identity, key)
		if err == nil {
			m.conn = t
			return nil
		}
	}
	if c.Code == "" {
		return errors.New("enter the IKEA gateway security code")
	}
	t, err := ikeaDial(c.Address, "Client_identity", c.Code)
	if err != nil {
		return err
	}
	identity = "home-controller-" + token()[:16]
	var response struct {
		Key string `json:"9091"`
	}
	err = t.request(codes.POST, "/15011/9063", map[string]string{"9090": identity}, &response)
	t.conn.Close()
	if err != nil || response.Key == "" {
		return errors.New("IKEA authentication failed; check the security code")
	}
	key = response.Key
	if err = atomicJSON(m.sessionPath, ikeaSession{fingerprint, identity, key}); err != nil {
		return errors.New("could not save IKEA connection credentials")
	}
	m.conn, err = ikeaDial(c.Address, identity, key)
	return err
}
func (m *ikeaManager) refresh() {
	m.network.Lock()
	defer m.network.Unlock()
	var c ikeaCredentials
	if err := readJSON(m.credentialsPath, &c); err != nil {
		m.close()
		m.publish(IKEAStatus{Targets: []IKEATargetInfo{}})
		return
	}
	fingerprint := hash(c.Address + "\x00" + c.Gateway + "\x00" + c.Code + "\x00" + c.Key)
	if m.fingerprint != fingerprint {
		m.close()
		m.fingerprint = fingerprint
		m.publish(IKEAStatus{Configured: true, Address: c.Address, Gateway: c.Gateway, Targets: []IKEATargetInfo{}})
	}
	state := m.snapshot()
	state.Configured = true
	state.Address = c.Address
	state.Gateway = c.Gateway
	fail := func(err error) { m.close(); state.Connected = false; state.Error = err.Error(); m.publish(state) }
	if m.conn == nil {
		if err := m.connect(c, fingerprint); err != nil {
			fail(err)
			return
		}
	}
	targets, err := readIKEAInventory(func(path string, out any) error { return m.conn.request(codes.GET, path, nil, out) }, c.Gateway)
	if err != nil {
		fail(errors.New("IKEA gateway unavailable"))
		return
	}
	state.Targets = targets
	state.Connected = true
	state.Error = ""
	m.publish(state)
}
func (m *ikeaManager) loop() {
	for {
		m.refresh()
		select {
		case <-time.After(15 * time.Second):
		case <-m.wake:
		}
	}
}

func rawInt(d map[string]json.RawMessage, key string) int {
	var n int
	json.Unmarshal(d[key], &n)
	return n
}
func rawString(d map[string]json.RawMessage, key string) string {
	var s string
	json.Unmarshal(d[key], &s)
	return s
}
func readIKEAInventory(get func(string, any) error, gateway string) ([]IKEATargetInfo, error) {
	var ids []int
	if err := get("/15001", &ids); err != nil {
		return nil, err
	}
	if len(ids) > 256 {
		return nil, errors.New("too many IKEA devices")
	}
	all := map[int]IKEATargetInfo{}
	out := []IKEATargetInfo{}
	for _, id := range ids {
		if id < 1 || id > 2147483647 {
			return nil, errors.New("invalid IKEA resource")
		}
		var raw map[string]json.RawMessage
		if err := get(fmt.Sprintf("/15001/%d", id), &raw); err != nil {
			return nil, err
		}
		t := ikeaDevice(raw, gateway, id)
		all[id] = t
		if t.Capabilities.Power {
			out = append(out, t)
		}
	}
	if err := get("/15004", &ids); err != nil {
		return nil, err
	}
	if len(ids) > 256 {
		return nil, errors.New("too many IKEA groups")
	}
	for _, id := range ids {
		if id < 1 || id > 2147483647 {
			return nil, errors.New("invalid IKEA resource")
		}
		var raw map[string]json.RawMessage
		if err := get(fmt.Sprintf("/15004/%d", id), &raw); err != nil {
			return nil, err
		}
		var members struct {
			Devices struct {
				IDs []int `json:"9003"`
			} `json:"15002"`
		}
		json.Unmarshal(raw["9018"], &members)
		t := IKEATargetInfo{IKEATarget: IKEATarget{Gateway: gateway, Kind: "group", ID: id}, Name: rawString(raw, "9001")}
		seen := map[int]bool{}
		available := 0
		on := 0
		for _, mid := range members.Devices.IDs {
			d, ok := all[mid]
			if !ok || !d.Capabilities.Power || seen[mid] {
				continue
			}
			seen[mid] = true
			t.Members = append(t.Members, mid)
			t.Capabilities.Power = true
			t.Capabilities.Brightness = t.Capabilities.Brightness || d.Capabilities.Brightness
			t.Capabilities.Temperature = t.Capabilities.Temperature || d.Capabilities.Temperature
			t.Capabilities.Color = t.Capabilities.Color || d.Capabilities.Color
			if d.Reachable {
				available++
				if d.Power != nil && *d.Power {
					on++
				}
			}
		}
		t.Reachable = available > 0
		t.Partial = available > 0 && available < len(t.Members)
		if available > 0 && (on == 0 || on == available) {
			power := on > 0
			t.Power = &power
		}
		// Group feedback is derived from actual members, never the gateway's stale group state.
		for _, mid := range t.Members {
			d := all[mid]
			if d.Reachable {
				if d.Brightness != nil {
					if t.Brightness == nil {
						v := *d.Brightness
						t.Brightness = &v
					} else if *t.Brightness != *d.Brightness {
						v := -1
						t.Brightness = &v
					}
				}
				if d.Temperature != nil {
					if t.Temperature == nil {
						v := *d.Temperature
						t.Temperature = &v
					} else if *t.Temperature != *d.Temperature {
						v := -1
						t.Temperature = &v
					}
				}
				if d.Color != "" {
					if t.Color == "" {
						t.Color = d.Color
					} else if t.Color != d.Color {
						t.Color = "mixed"
					}
				}
			}
		}
		if t.Capabilities.Power {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
func ikeaDevice(raw map[string]json.RawMessage, gateway string, id int) IKEATargetInfo {
	t := IKEATargetInfo{IKEATarget: IKEATarget{Gateway: gateway, Kind: "device", ID: id}, Name: rawString(raw, "9001"), Reachable: rawInt(raw, "9019") == 1}
	var info map[string]json.RawMessage
	json.Unmarshal(raw["3"], &info)
	t.Model = rawString(info, "1")
	kind := rawInt(raw, "5750")
	var controls []map[string]json.RawMessage
	if kind == 2 {
		json.Unmarshal(raw["3311"], &controls)
	} else if kind == 3 {
		json.Unmarshal(raw["3312"], &controls)
	}
	if len(controls) == 0 {
		return t
	}
	d := controls[0]
	_, t.Capabilities.Power = d["5850"]
	power := rawInt(d, "5850") == 1
	t.Power = &power
	if kind == 2 {
		_, t.Capabilities.Brightness = d["5851"]
		_, t.Capabilities.Temperature = d["5711"]
		_, h := d["5707"]
		_, s := d["5708"]
		t.Capabilities.Color = h && s
		if t.Capabilities.Brightness {
			v := int(math.Round(float64(rawInt(d, "5851")) * 100 / 254))
			t.Brightness = &v
		}
		if t.Capabilities.Temperature && rawInt(d, "5711") > 0 {
			v := int(math.Round(1e6/float64(rawInt(d, "5711"))/50) * 50)
			t.Temperature = &v
		}
		if t.Capabilities.Color {
			t.Color = ikeaHSVHex(float64(rawInt(d, "5707"))/65535*360, float64(rawInt(d, "5708"))/65279)
		}
	}
	return t
}
func validateActionValue(action string, v ActionValue) error {
	switch action {
	case "brightness":
		if v.Value == nil || *v.Value < 0 || *v.Value > 100 || v.Color != "" {
			return errors.New("brightness must be 0 to 100 percent")
		}
	case "temperature":
		if v.Value == nil || *v.Value < 2200 || *v.Value > 4000 || v.Color != "" {
			return errors.New("white temperature must be 2200 to 4000 K")
		}
	case "color":
		if v.Value != nil || len(v.Color) != 7 || v.Color[0] != '#' {
			return errors.New("choose a colour")
		}
		if _, e := strconv.ParseUint(v.Color[1:], 16, 24); e != nil {
			return errors.New("invalid colour")
		}
	default:
		if v.Value != nil || v.Color != "" {
			return errors.New("this action does not accept a value")
		}
	}
	return nil
}
func ikeaPayload(t IKEATargetInfo, action string, v ActionValue) (map[string]any, error) {
	if !t.Capabilities.supports(action) {
		return nil, errors.New("action not supported by IKEA target")
	}
	if err := validateActionValue(action, v); err != nil {
		return nil, err
	}
	p := map[string]any{}
	switch action {
	case "on":
		p["5850"] = 1
	case "off":
		p["5850"] = 0
	case "brightness":
		p["5851"] = int(math.Round(float64(*v.Value) * 254 / 100))
		if *v.Value == 0 {
			p["5850"] = 0
		} else {
			p["5850"] = 1
		}
	case "temperature":
		p["5711"] = int(math.Round(1e6 / float64(*v.Value)))
		p["5850"] = 1
	case "color":
		h, s := ikeaHexHSV(v.Color)
		p["5707"] = int(math.Round(h / 360 * 65535))
		p["5708"] = int(math.Round(s * 65279))
		p["5850"] = 1
	}
	key := "3311"
	if !t.Capabilities.Brightness {
		key = "3312"
	}
	return map[string]any{key: []any{p}}, nil
}
func (m *ikeaManager) send(target IKEATarget, action string, v ActionValue, expires time.Time) error {
	m.network.Lock()
	defer m.network.Unlock()
	state := m.snapshot()
	if !state.Connected || m.conn == nil {
		return errors.New("IKEA gateway unavailable")
	}
	var selected *IKEATargetInfo
	for i := range state.Targets {
		t := &state.Targets[i]
		if t.Gateway == target.Gateway && t.Kind == target.Kind && t.ID == target.ID {
			selected = t
			break
		}
	}
	if selected == nil || !selected.Reachable {
		return errors.New("IKEA target unavailable")
	}
	ids := []int{selected.ID}
	if selected.Kind == "group" {
		ids = selected.Members
	}
	sent, failed := 0, 0
	for _, id := range ids {
		for _, t := range state.Targets {
			if t.Kind != "device" || t.ID != id || !t.Capabilities.supports(action) {
				continue
			}
			if !t.Reachable || (!expires.IsZero() && time.Now().After(expires)) {
				failed++
				continue
			}
			body, err := ikeaPayload(t, action, v)
			if err != nil {
				return err
			}
			if err = m.conn.request(codes.PUT, fmt.Sprintf("/15001/%d", id), body, nil); err != nil {
				m.close()
				state.Connected = false
				state.Error = "IKEA gateway unavailable"
				m.publish(state)
				select {
				case m.wake <- true:
				default:
				}
				return fmt.Errorf("IKEA command failed after %d successful sends", sent)
			} else {
				sent++
			}
		}
	}
	select {
	case m.wake <- true:
	default:
	}
	if failed > 0 {
		return fmt.Errorf("IKEA: %d sent, %d unavailable or failed", sent, failed)
	}
	if sent == 0 {
		return errors.New("no IKEA members support this action")
	}
	return nil
}
func (s *Server) ikeaTarget(d Device) IKEATargetInfo {
	t := IKEATargetInfo{}
	if d.IKEA != nil {
		t.IKEATarget = *d.IKEA
	}
	if s.ikea == nil {
		return t
	}
	state := s.ikea.snapshot()
	for _, live := range state.Targets {
		if live.Gateway == t.Gateway && live.Kind == t.Kind && live.ID == t.ID {
			if !state.Connected {
				live.Reachable = false
			}
			return live
		}
	}
	return t
}
func ikeaHexHSV(color string) (float64, float64) {
	n, _ := strconv.ParseUint(strings.TrimPrefix(color, "#"), 16, 24)
	r, g, b := float64(n>>16)/255, float64((n>>8)&255)/255, float64(n&255)/255
	max, min := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	delta := max - min
	if delta == 0 {
		return 0, 0
	}
	h := 0.0
	switch max {
	case r:
		h = math.Mod((g-b)/delta, 6)
	case g:
		h = (b-r)/delta + 2
	case b:
		h = (r-g)/delta + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, delta / max
}
func ikeaHSVHex(h, s float64) string {
	s = math.Max(0, math.Min(1, s))
	x := s * (1 - math.Abs(math.Mod(h/60, 2)-1))
	r, g, b := 0.0, 0.0, 0.0
	switch {
	case h < 60:
		r, g = s, x
	case h < 120:
		r, g = x, s
	case h < 180:
		g, b = s, x
	case h < 240:
		g, b = x, s
	case h < 300:
		r, b = x, s
	default:
		r, b = s, x
	}
	m := 1 - s
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round((r+m)*255)), int(math.Round((g+m)*255)), int(math.Round((b+m)*255)))
}

func ikeaSettings() IKEAStatus {
	var v IKEAStatus
	readJSON(run+"/ikea.json", &v)
	var c ikeaCredentials
	if readJSON(base+"/ikea/credentials.json", &c) == nil {
		v.Configured = true
		v.Address = c.Address
		v.Gateway = c.Gateway
	}
	return v
}
func prepareIKEACredentials() error {
	path := base + "/ikea/credentials.json"
	if !exists(path) {
		return nil
	}
	adminFile(base + "/ikea")
	if err := os.Chmod(base+"/ikea", 0750); err != nil {
		return err
	}
	adminFile(path)
	return nil
}
func saveIKEACredentials(address, code string) error {
	var c ikeaCredentials
	readJSON(base+"/ikea/credentials.json", &c)
	if address == "" {
		address = c.Address
	}
	ip := net.ParseIP(address)
	if ip == nil || !ip.IsPrivate() || ip.IsLoopback() {
		return errors.New("enter the gateway's private IP address")
	}
	if code == "" {
		code = c.Code
	}
	if len(code) < 8 || len(code) > 128 || strings.ContainsAny(code, "\r\n\x00") {
		return errors.New("enter the printed gateway security code")
	}
	// An explicit replacement creates a new gateway identity, preventing old tiles
	// from silently controlling a different gateway that reused numeric resource IDs.
	if c.Gateway == "" || c.Code != code {
		c.Gateway = "ikea-" + token()[:16]
	}
	c.Address = ip.String()
	c.Code = code
	c.Identity = ""
	c.Key = ""
	if err := atomicJSON(base+"/ikea/credentials.json", c); err != nil {
		return err
	}
	os.Remove(base + "/state/ikea-auth.json")
	os.Remove(run + "/ikea.json")
	return prepareIKEACredentials()
}
