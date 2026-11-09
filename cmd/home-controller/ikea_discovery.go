package main

import (
	"context"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type ikeaLease struct{ MAC, Address, Hostname string }

func ikeaLeases(data string, now time.Time) []ikeaLease {
	out := []ikeaLease{}
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		expiry, err := strconv.ParseInt(f[0], 10, 64)
		if err != nil || (expiry != 0 && expiry < now.Unix()) {
			continue
		}
		mac, err := net.ParseMAC(f[1])
		if err != nil || len(mac) != 6 {
			continue
		}
		ip := net.ParseIP(f[2])
		if ip == nil || !ip.IsPrivate() || ip.IsLoopback() {
			continue
		}
		host := f[3]
		if !ikeaHostname(host) {
			host = ""
		}
		out = append(out, ikeaLease{strings.ToLower(mac.String()), ip.String(), host})
	}
	return out
}
func ikeaHostname(s string) bool {
	if len(s) < 1 || len(s) > 253 || net.ParseIP(s) != nil {
		return false
	}
	for _, part := range strings.Split(s, ".") {
		if len(part) < 1 || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
func currentIKEALeases() []ikeaLease {
	b, _ := os.ReadFile("/tmp/dhcp.leases")
	return ikeaLeases(string(b), time.Now())
}
func pinIKEAGateway(c *ikeaCredentials, leases []ikeaLease) bool {
	if c.MAC != "" {
		return false
	}
	for _, l := range leases {
		if l.Address == c.Address {
			c.MAC = l.MAC
			c.Hostname = l.Hostname
			return true
		}
	}
	return false
}
func lookupIKEAHost(name string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return net.DefaultResolver.LookupIP(ctx, "ip4", name)
}
func resolveIKEAGateway(c ikeaCredentials, leases []ikeaLease, lookup func(string) ([]net.IP, error)) (string, error) {
	if c.MAC != "" {
		for _, l := range leases {
			if strings.EqualFold(l.MAC, c.MAC) {
				return l.Address, nil
			}
		}
	}
	// DHCP is authoritative for the pinned MAC. A local hostname is a fallback,
	// never a replacement identity; the DTLS key still authenticates the gateway.
	host := c.Hostname
	if host == "" && ikeaHostname(c.Address) {
		host = c.Address
	}
	if host != "" {
		ips, err := lookup(host)
		if err == nil {
			for _, ip := range ips {
				if !ip.IsPrivate() || ip.IsLoopback() {
					continue
				}
				conflict := false
				for _, l := range leases {
					if l.Address == ip.String() && c.MAC != "" && !strings.EqualFold(l.MAC, c.MAC) {
						conflict = true
					}
				}
				if !conflict {
					return ip.String(), nil
				}
			}
		}
	}
	if ip := net.ParseIP(c.Address); ip != nil && ip.IsPrivate() && !ip.IsLoopback() {
		for _, l := range leases {
			if l.Address == ip.String() && c.MAC != "" && !strings.EqualFold(l.MAC, c.MAC) {
				return "", errors.New("IKEA gateway address changed; waiting for discovery")
			}
		}
		return ip.String(), nil
	}
	return "", errors.New("IKEA gateway address unavailable")
}

func ikeaFingerprint(c ikeaCredentials) string {
	return hash(c.Gateway + "\x00" + c.Code + "\x00" + c.Key)
}
