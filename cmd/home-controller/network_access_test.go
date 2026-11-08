package main

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"
)

func TestNetworkSelectionsControlListenersAndIngress(t *testing.T) {
	var interfaces []iface
	err := json.Unmarshal([]byte(`[
	 {"interface":"lan","l3_device":"br-lan","up":true,"ipv4-address":[{"address":"192.0.2.1"}]},
	 {"interface":"vpn","proto":"wireguard","l3_device":"wg_test","up":true,"ipv4-address":[{"address":"198.51.100.1"}]},
	 {"interface":"openvpn","proto":"openvpn","l3_device":"tun0","up":true},
	 {"interface":"openconnect","proto":"openconnect","l3_device":"vpn0","up":true},
	 {"interface":"tap","l3_device":"tap0","up":true},
	 {"interface":"ipsec","l3_device":"xfrm0","up":true},
	 {"interface":"offline","proto":"wireguard","l3_device":"wg_down","up":false},
	 {"interface":"wan","l3_device":"eth1","up":true,"ipv4-address":[{"address":"203.0.113.1"}]},
	 {"interface":"wan6","l3_device":"eth2","up":true},
	 {"interface":"alias","l3_device":"eth1","up":true}
	]`), &interfaces)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		selected []string
		devices  []string
		binds    []string
		wantErr  bool
	}{
		{"LAN excludes all implicit VPNs", []string{"lan"}, []string{"lo", "br-lan"}, []string{"192.0.2.1:8443", "[2001:db8::1]:8443"}, false},
		{"selected VPN included", []string{"lan", "vpn"}, []string{"lo", "br-lan", "wg_test"}, []string{"192.0.2.1:8443", "198.51.100.1:8443", "[2001:db8::1]:8443"}, false},
		{"VPN only excludes LAN", []string{"vpn"}, []string{"lo", "wg_test"}, []string{"198.51.100.1:8443"}, false},
		{"offline VPN excluded", []string{"lan", "offline"}, []string{"lo", "br-lan"}, []string{"192.0.2.1:8443", "[2001:db8::1]:8443"}, false},
		{"selected TUN with kernel-only IPv6", []string{"openvpn"}, []string{"lo", "tun0"}, []string{"[2001:db8:1::1]:8443"}, false},
		{"WAN rejected", []string{"lan", "wan"}, nil, nil, true},
		{"WAN6 rejected", []string{"wan6"}, nil, nil, true},
		{"WAN device alias rejected", []string{"alias"}, nil, nil, true},
		{"offline-only selection rejected", []string{"offline"}, nil, nil, true},
		{"empty selection rejected", nil, nil, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := exampleConfig()
			c.Networks = tc.selected
			devices, binds, err := networkAccessForInterfaces(c, interfaces, func(name string) ([]net.Addr, error) {
				if !contains(tc.devices, name) {
					t.Fatalf("looked up unselected interface %q", name)
				}
				address := "fe80::1/64"
				if name == "br-lan" {
					address = "2001:db8::1/64"
				} else if name == "tun0" {
					address = "2001:db8:1::1/64"
				}
				ip, network, _ := net.ParseCIDR(address)
				network.IP = ip
				return []net.Addr{network}, nil
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, want error %v", err, tc.wantErr)
			}
			if !tc.wantErr && (!reflect.DeepEqual(devices, tc.devices) || !reflect.DeepEqual(binds, tc.binds)) {
				t.Fatalf("got devices %v, binds %v; want %v, %v", devices, binds, tc.devices, tc.binds)
			}
		})
	}
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
