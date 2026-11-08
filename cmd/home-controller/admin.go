package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func own(path string) {
	u, e := user.Lookup("homecontroller")
	if e != nil {
		return
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)
	os.Chown(path, uid, gid)
}
func rpc() error {
	if os.Geteuid() != 0 {
		return errors.New("administrator required")
	}
	if len(os.Args) > 2 && os.Args[2] == "list" {
		fmt.Print(`{"get":{},"save":{"config":""},"password":{"user":"","password":""},"revoke":{},"sessions":{}}`)
		return nil
	}
	if len(os.Args) < 4 {
		return errors.New("missing RPC method")
	}
	method := os.Args[3]
	respond := func(v any) { json.NewEncoder(os.Stdout).Encode(v) }
	errResult := func(e error) error { respond(map[string]string{"error": e.Error()}); return nil }
	switch method {
	case "get":
		c, e := loadConfig()
		if e != nil {
			return errResult(e)
		}
		var status any
		readJSON(run+"/status.json", &status)
		var cred Credential
		readJSON(base+"/credential.json", &cred)
		var access any
		readJSON(run+"/access.json", &access)
		respond(map[string]any{"config": c, "status": status, "account": cred.User, "access": access, "sunwait": exists("/usr/bin/sunwait")})
	case "save":
		var args struct {
			Config string `json:"config"`
		}
		if e := decode(os.Stdin, &args); e != nil {
			return errResult(e)
		}
		var c Config
		if e := decode(strings.NewReader(args.Config), &c); e != nil {
			return errResult(e)
		}
		if e := validate(c); e != nil {
			return errResult(e)
		}
		if _, _, e := networkAccess(c); e != nil {
			return errResult(e)
		}
		if e := atomicJSON(base+"/config.json", c); e != nil {
			return errResult(e)
		}
		own(base + "/config.json")
		restart()
		respond(map[string]bool{"ok": true})
	case "password":
		var args struct {
			User     string `json:"user"`
			Password string `json:"password"`
		}
		if e := decode(os.Stdin, &args); e != nil {
			return errResult(e)
		}
		if !idRE.MatchString(args.User) || len(args.Password) < 12 || len(args.Password) > 256 {
			return errResult(errors.New("username: lowercase letters/numbers; password: 12..256 characters"))
		}
		c := Credential{User: args.User, Salt: token(), Revision: token()}
		c.Hash = passwordHash(args.Password, c.Salt)
		if e := atomicJSON(base+"/credential.json", c); e != nil {
			return errResult(e)
		}
		own(base + "/credential.json")
		atomicJSON(base+"/state/sessions.json", []Session{})
		own(base + "/state/sessions.json")
		restart()
		respond(map[string]bool{"ok": true})
	case "revoke":
		var c Credential
		if e := readJSON(base+"/credential.json", &c); e != nil {
			return errResult(e)
		}
		c.Revision = token()
		atomicJSON(base+"/credential.json", c)
		own(base + "/credential.json")
		atomicJSON(base+"/state/sessions.json", []Session{})
		own(base + "/state/sessions.json")
		restart()
		respond(map[string]bool{"ok": true})
	case "sessions":
		var sessions []Session
		readJSON(base+"/state/sessions.json", &sessions)
		out := []map[string]any{}
		for _, s := range sessions {
			if s.Expires > time.Now().Unix() {
				out = append(out, map[string]any{"created": s.Created, "expires": s.Expires, "device": s.Device})
			}
		}
		respond(map[string]any{"sessions": out})
	default:
		return errors.New("unknown RPC method")
	}
	return nil
}
func restart() {
	cmd := exec.Command("/etc/init.d/home-controller", "restart")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Start()
}
func exists(p string) bool { _, e := os.Stat(p); return e == nil }

type iface struct {
	Interface string `json:"interface"`
	Proto     string `json:"proto"`
	Device    string `json:"l3_device"`
	Up        bool   `json:"up"`
	Addresses []struct {
		Address string `json:"address"`
	} `json:"ipv4-address"`
	IPv6 []struct {
		Address string `json:"address"`
	} `json:"ipv6-address"`
}

func networkAccess(c Config) ([]string, []string, error) {
	out, e := exec.Command("/bin/ubus", "call", "network.interface", "dump").Output()
	if e != nil {
		return nil, nil, e
	}
	var data struct {
		Interface []iface `json:"interface"`
	}
	if e = json.Unmarshal(out, &data); e != nil {
		return nil, nil, e
	}
	denied := map[string]bool{}
	for _, i := range data.Interface {
		if i.Interface == "wan" || i.Interface == "wan6" {
			denied[i.Device] = true
		}
	}
	devices := []string{"lo"}
	binds := []string{}
	want := map[string]bool{}
	for _, n := range c.Networks {
		want[n] = true
	}
	for _, i := range data.Interface {
		vpn := i.Proto == "wireguard" || i.Proto == "openvpn" || i.Proto == "openconnect" || strings.HasPrefix(i.Device, "tun") || strings.HasPrefix(i.Device, "tap") || strings.HasPrefix(i.Device, "xfrm")
		if !i.Up || (!want[i.Interface] && !vpn) {
			continue
		}
		if denied[i.Device] || !regexpDevice(i.Device) {
			return nil, nil, errors.New("selected network uses WAN or invalid interface")
		}
		devices = append(devices, i.Device)
		for _, a := range i.Addresses {
			if net.ParseIP(a.Address) != nil {
				binds = append(binds, net.JoinHostPort(a.Address, strconv.Itoa(c.Port)))
			}
		}
		for _, a := range i.IPv6 {
			if ip := net.ParseIP(a.Address); ip != nil && !ip.IsLinkLocalUnicast() {
				binds = append(binds, net.JoinHostPort(a.Address, strconv.Itoa(c.Port)))
			}
		}
	}
	// OpenVPN can create tun/tap interfaces outside netifd. Include active ones.
	links, _ := net.Interfaces()
	for _, i := range links {
		if i.Flags&net.FlagUp != 0 && !denied[i.Name] && (strings.HasPrefix(i.Name, "tun") || strings.HasPrefix(i.Name, "tap") || strings.HasPrefix(i.Name, "xfrm")) && regexpDevice(i.Name) {
			devices = append(devices, i.Name)
			addresses, _ := i.Addrs()
			for _, a := range addresses {
				ip, _, e := net.ParseCIDR(a.String())
				if e == nil && !ip.IsLinkLocalUnicast() {
					binds = append(binds, net.JoinHostPort(ip.String(), strconv.Itoa(c.Port)))
				}
			}
		}
	}
	if len(binds) == 0 {
		return nil, nil, errors.New("no permitted LAN/VPN address is active")
	}
	return devices, binds, nil
}
func regexpDevice(s string) bool {
	if len(s) < 1 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_.-", r)) {
			return false
		}
	}
	return true
}
func configure() error {
	if os.Geteuid() != 0 {
		return errors.New("root required")
	}
	c, e := loadConfig()
	if e != nil {
		return e
	}
	devices, binds, e := networkAccess(c)
	if e != nil {
		return e
	}
	for _, dir := range []string{base, base + "/state", run} {
		os.MkdirAll(dir, 0700)
		own(dir)
	}
	for _, p := range []string{"config.json", "credential.json", "state/sessions.json"} {
		own(base + "/" + p)
	}
	// Dedicated firewall table is evaluated before the regular router input chains.
	exec.Command("/usr/sbin/nft", "delete", "table", "inet", "home_controller").Run()
	quoted := []string{}
	for _, d := range devices {
		quoted = append(quoted, strconv.Quote(d))
	}
	rules := fmt.Sprintf("table inet home_controller { chain input { type filter hook input priority -10; policy accept; tcp dport %d iifname != { %s } drop; } }\n", c.Port, strings.Join(quoted, ", "))
	cmd := exec.Command("/usr/sbin/nft", "-f", "-")
	cmd.Stdin = strings.NewReader(rules)
	if out, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("firewall: %s", out)
	}
	// Independent document root: no LuCI, ubus proxy, or arbitrary CGI scripts.
	args := []string{"-f", "-h", "/usr/share/home-controller/www", "-D", "-S", "-x", "/api", "-t", "20", "-T", "20", "-n", "3", "-N", "24", "-C", base + "/tls.crt", "-K", base + "/tls.key"}
	for _, b := range binds {
		args = append(args, "-s", b)
	}
	if e = atomicJSON(run+"/listen.json", args); e != nil {
		return e
	}
	own(run + "/listen.json")
	atomicJSON(run+"/access.json", map[string]any{"interfaces": devices, "listeners": binds})
	own(run + "/access.json")
	// Do not override another application's TellStick configuration silently.
	var radio strings.Builder
	radio.WriteString("# Managed by Home Controller\nuser = \"homecontroller\"\ngroup = \"homecontroller\"\n")
	if c.Controller != "" {
		fmt.Fprintf(&radio, "controller { id = 1; type = 1; serial = %q; }\n", c.Controller)
	}
	for _, d := range c.Devices {
		fmt.Fprintf(&radio, "device { id = %d; name = %q; protocol = \"hasta\"; model = \"selflearningv2\"; controller = 1; parameters { house = %d; unit = %d; } }\n", d.RadioID, d.ID, d.House, d.Unit)
	}
	if e = os.WriteFile("/etc/tellstick.conf", []byte(radio.String()), 0600); e != nil {
		return e
	}
	own("/etc/tellstick.conf")
	for _, p := range []string{base + "/tls.crt", base + "/tls.key"} {
		if !exists(p) {
			return errors.New("configure TLS certificate and key before enabling")
		}
		os.Chmod(p, 0600)
		own(p)
	}
	// Generate a shell argument file with validated addresses and fixed options only.
	lines := []string{}
	for _, a := range args {
		lines = append(lines, "'"+strings.ReplaceAll(a, "'", "'\\''")+"'")
	}
	p := filepath.Join(run, "uhttpd.args")
	e = os.WriteFile(p, []byte(strings.Join(lines, " ")), 0600)
	own(p)
	return e
}
