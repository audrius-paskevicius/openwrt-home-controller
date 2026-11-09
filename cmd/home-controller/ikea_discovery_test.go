package main

import (
	"net"
	"testing"
	"time"
)

func TestIKEADHCPAddressTracking(t *testing.T) {
	now := time.Unix(100, 0)
	leasing := "200 02:00:00:00:00:01 192.168.20.9 hub-test *\n200 02:00:00:00:00:02 192.168.20.8 other *\n50 02:00:00:00:00:01 192.168.20.4 old *\n200 bad 192.168.20.3 bad *\n200 02:00:00:00:00:03 8.8.8.8 public *\n"
	leases := ikeaLeases(leasing, now)
	if len(leases) != 2 {
		t.Fatal(leases)
	}
	c := ikeaCredentials{Address: "192.168.20.9", Gateway: "test", Code: "test-only-code"}
	if !pinIKEAGateway(&c, leases) || c.MAC != "02:00:00:00:00:01" || c.Hostname != "hub-test" {
		t.Fatal(c.MAC, c.Hostname)
	}
	fingerprint := ikeaFingerprint(c)
	c.Address = "192.168.20.8"
	if pinIKEAGateway(&c, leases) {
		t.Fatal("replaced a pinned MAC")
	}
	lookup := func(string) ([]net.IP, error) { t.Fatal("DHCP match should not need DNS"); return nil, nil }
	if ip, err := resolveIKEAGateway(c, leases, lookup); err != nil || ip != "192.168.20.9" {
		t.Fatal(ip, err)
	}
	if ikeaFingerprint(c) != fingerprint {
		t.Fatal("address change invalidated authentication")
	}
	lookup = func(host string) ([]net.IP, error) {
		if host != "hub-test" {
			t.Fatal(host)
		}
		return []net.IP{net.ParseIP("192.168.20.7")}, nil
	}
	if ip, err := resolveIKEAGateway(c, nil, lookup); err != nil || ip != "192.168.20.7" {
		t.Fatal(ip, err)
	}
	lookup = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("127.0.0.1"), net.ParseIP("192.168.20.8")}, nil
	}
	if _, err := resolveIKEAGateway(c, leases[1:], lookup); err == nil {
		t.Fatal("accepted a reassigned IP for another MAC")
	}
	for _, host := range []string{"../secret", "host:5684", "bad name", "-host", "host..lan", "a/b"} {
		if ikeaHostname(host) {
			t.Fatal(host)
		}
	}
}
func TestIKEAHueSaturationAndMode(t *testing.T) {
	d := ikeaTestDevice()
	target := IKEATargetInfo{IKEATarget: *d.IKEA, Reachable: true, colorSignature: "initial", Hue: intPointer(80), Saturation: intPointer(25), rawHue: 14564, rawSaturation: 16320}
	for _, tc := range []struct {
		action, key string
		value, want int
	}{{"hue", "5707", 120, 21845}, {"saturation", "5708", 25, 16320}} {
		body, err := ikeaPayload(target, tc.action, ActionValue{Value: &tc.value})
		if err != nil {
			t.Fatal(err)
		}
		p := body["3311"].([]any)[0].(map[string]any)
		if p[tc.key] != tc.want || len(p) != 3 || (tc.action == "hue" && p["5708"] != target.rawSaturation) || (tc.action == "saturation" && p["5707"] != target.rawHue) {
			t.Fatal("partial colour update must preserve the other setting and brightness", p)
		}
	}
	for _, tc := range []struct {
		action string
		value  int
	}{{"hue", 360}, {"saturation", 101}, {"hue", -1}} {
		if validateActionValue(tc.action, ActionValue{Value: &tc.value}) == nil {
			t.Fatal(tc)
		}
	}
	m := &ikeaManager{}
	m.recordMode(target, "temperature")
	items := []IKEATargetInfo{target}
	m.applyModes(items)
	if items[0].Mode != "white" {
		t.Fatal("white mode missing")
	}
	key := ikeaModeKey(target)
	r := m.modes[key]
	r.At = time.Now().Add(-time.Minute).Unix()
	m.modes[key] = r
	items[0].colorSignature = "external change"
	items[0].Mode = ""
	m.applyModes(items)
	if items[0].Mode != "" {
		t.Fatal("external colour change retained misleading white mode")
	}
	m.recordMode(target, "saturation")
	items = []IKEATargetInfo{target}
	m.applyModes(items)
	if items[0].Mode != "color" {
		t.Fatal("saturation did not select colour")
	}
}
