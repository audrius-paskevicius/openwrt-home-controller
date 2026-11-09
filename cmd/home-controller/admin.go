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
	"reflect"
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
	os.Lchown(path, uid, gid)
}

func adminFile(path string) {
	u, e := user.Lookup("homecontroller")
	if e != nil {
		return
	}
	gid, _ := strconv.Atoi(u.Gid)
	os.Lchown(path, 0, gid)
	os.Chmod(path, 0640)
}
func rpc() error {
	if os.Geteuid() != 0 {
		return errors.New("administrator required")
	}
	if len(os.Args) > 2 && os.Args[2] == "list" {
		fmt.Print(`{"get":{},"save":{"config":"","controller_instance":""},"stage":{"config":"","controller_instance":""},"discard":{},"password":{"user":"","password":""},"revoke":{},"sessions":{},"tls":{},"ikea":{"address":"","code":""}}`)
		return nil
	}
	if len(os.Args) < 4 {
		return errors.New("missing RPC method")
	}
	method := os.Args[3]
	if method == "get" || method == "save" || method == "stage" || method == "discard" {
		unlock, err := configLock()
		if err != nil {
			return err
		}
		defer unlock()
	}
	respond := func(v any) { json.NewEncoder(os.Stdout).Encode(v) }
	errResult := func(e error) error { respond(map[string]string{"error": e.Error()}); return nil }
	switch method {
	case "ikea":
		var args struct {
			Address    string `json:"address"`
			Code       string `json:"code"`
			RPCSession string `json:"ubus_rpc_session"`
		}
		if e := decode(os.Stdin, &args); e != nil {
			return errResult(e)
		}
		unlock, e := configLock()
		if e != nil {
			return errResult(e)
		}
		defer unlock()
		if e = saveIKEACredentials(args.Address, args.Code); e != nil {
			return errResult(e)
		}
		restart()
		respond(map[string]bool{"ok": true})
	case "tls":
		if e := renewTLS(); e != nil {
			return errResult(e)
		}
		restart()
		respond(map[string]bool{"ok": true})
	case "get":
		available := controllers()
		if e := clearMissingPreferences(base, available); e != nil {
			return errResult(e)
		}
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
		var active map[string]string
		readJSON(run+"/controller.json", &active)
		connected := false
		for _, usb := range available {
			if usb["serial"] == active["serial"] {
				connected = true
			}
		}
		if !c.Enabled || !connected {
			active = nil
		}
		editable, pending, e := editableConfig(base, c)
		if e != nil {
			return errResult(e)
		}
		respond(map[string]any{"ikea": ikeaSettings(), "config": editable, "active_config": c, "pending": pending, "status": status, "account": cred.User, "access": access, "sunwait": exists("/usr/bin/sunwait"), "controllers": available, "active_controller": active, "networks": availableNetworks(), "time_utc": time.Now().UTC().Format(time.RFC3339)})
	case "discard":
		if e := discardConfig(base); e != nil {
			return errResult(e)
		}
		respond(map[string]bool{"ok": true})
	case "save", "stage":
		var args struct {
			Config             string `json:"config"`
			ControllerInstance string `json:"controller_instance"`
			RPCSession         string `json:"ubus_rpc_session"`
		}
		if e := decode(os.Stdin, &args); e != nil {
			return errResult(e)
		}
		c, e := parseConfig(strings.NewReader(args.Config))
		if e != nil {
			return errResult(e)
		}
		if _, _, e := networkAccess(c); e != nil {
			return errResult(e)
		}
		previous, e := loadConfig()
		if e != nil {
			return errResult(e)
		}
		c.PreferredController = submittedPreference(c.PreferredController, args.ControllerInstance, controllers())
		onlyPreference := previous
		onlyPreference.PreferredController = c.PreferredController
		if e := storeConfig(base, c, method == "save"); e != nil {
			return errResult(e)
		}
		if method == "save" {
			adminFile(base + "/config.json")
			if previous.PreferredController != c.PreferredController && reflect.DeepEqual(onlyPreference, c) {
				if e := refreshRadio(c); e != nil {
					return errResult(e)
				}
			} else {
				restart()
			}
		}
		c.PreferredController = connectedPreference(c.PreferredController, controllers())
		respond(map[string]any{"ok": true, "preferred_controller": c.PreferredController})
	case "password":
		var args struct {
			User       string `json:"user"`
			Password   string `json:"password"`
			RPCSession string `json:"ubus_rpc_session"`
		}
		if e := decode(os.Stdin, &args); e != nil {
			return errResult(e)
		}
		if e := validateSharedAccount(args.User, args.Password); e != nil {
			return errResult(e)
		}
		c := Credential{User: args.User, Salt: token(), Revision: token()}
		c.Hash = passwordHash(args.Password, c.Salt)
		if e := atomicJSON(base+"/credential.json", c); e != nil {
			return errResult(e)
		}
		adminFile(base + "/credential.json")
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
		adminFile(base + "/credential.json")
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

func controllers() []map[string]string {
	out := []map[string]string{}
	paths, _ := filepath.Glob("/sys/bus/usb/devices/*")
	for _, p := range paths {
		read := func(name string) string {
			b, _ := os.ReadFile(filepath.Join(p, name))
			return strings.TrimSpace(string(b))
		}
		if read("idVendor") != "1781" {
			continue
		}
		pid := read("idProduct")
		if pid != "0c30" && pid != "0c31" {
			continue
		}
		capability := "Transmit only"
		if pid == "0c31" {
			capability = "Transmit and receive"
		}
		realPath, _ := filepath.EvalSymlinks(p)
		out = append(out, map[string]string{"serial": read("serial"), "product": read("product"), "bus": read("busnum"), "device": read("devnum"), "capability": capability, "path": strings.TrimPrefix(realPath, "/sys")})
	}
	return out
}

// Keep the automatically chosen device while it is connected. With no previous
// device, sysfs enumeration provides a stable USB-port order. Never require a
// control administrator to bind configuration to a particular serial number.
func selectController(available []map[string]string, previous string) map[string]string {
	for _, usb := range available {
		if previous != "" && usb["serial"] == previous {
			return usb
		}
	}
	for _, usb := range available {
		if usb["serial"] != "" {
			return usb
		}
	}
	return nil
}

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

func availableNetworks() []string {
	out := []string{}
	raw, err := exec.Command("/bin/ubus", "call", "network.interface", "dump").Output()
	if err != nil {
		return out
	}
	var data struct {
		Interface []iface `json:"interface"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return out
	}
	denied := map[string]bool{}
	for _, i := range data.Interface {
		if i.Interface == "wan" || i.Interface == "wan6" {
			denied[i.Device] = true
		}
	}
	for _, i := range data.Interface {
		if i.Interface != "wan" && i.Interface != "wan6" && i.Device != "lo" && !denied[i.Device] && idRE.MatchString(i.Interface) {
			out = append(out, i.Interface)
		}
	}
	return out
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
	return networkAccessForInterfaces(c, data.Interface, func(name string) ([]net.Addr, error) {
		i, err := net.InterfaceByName(name)
		if err != nil {
			return nil, err
		}
		return i.Addrs()
	})
}

// Only explicit logical network selections may supply listeners or ingress devices.
// The address lookup also covers IPv6 prefix assignments omitted by netifd.
func networkAccessForInterfaces(c Config, interfaces []iface, addresses func(string) ([]net.Addr, error)) ([]string, []string, error) {
	denied := map[string]bool{}
	for _, i := range interfaces {
		if i.Interface == "wan" || i.Interface == "wan6" {
			denied[i.Device] = true
		}
	}
	devices := []string{"lo"}
	binds := []string{}
	want := map[string]bool{}
	for _, n := range c.Networks {
		if n == "wan" || n == "wan6" {
			return nil, nil, errors.New("WAN access is not allowed")
		}
		want[n] = true
	}
	for _, i := range interfaces {
		if !i.Up || !want[i.Interface] {
			continue
		}
		if denied[i.Device] || i.Device == "lo" || !regexpDevice(i.Device) {
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
	// Include IPv6 prefix-assigned LAN addresses, which netifd does not list
	// in ipv6-address. Never bind a device not already admitted above.
	for _, name := range devices {
		if name == "lo" {
			continue
		}
		addrs, e := addresses(name)
		if e != nil {
			continue
		}
		for _, a := range addrs {
			ip, _, e := net.ParseCIDR(a.String())
			if e == nil && !ip.IsLinkLocalUnicast() {
				binds = append(binds, net.JoinHostPort(ip.String(), strconv.Itoa(c.Port)))
			}
		}
	}
	if len(binds) == 0 {
		return nil, nil, errors.New("no selected LAN/VPN network has an active address")
	}
	unique := func(items []string) []string {
		out := []string{}
		seen := map[string]bool{}
		for _, v := range items {
			if !seen[v] {
				out = append(out, v)
				seen[v] = true
			}
		}
		return out
	}
	devices = unique(devices)
	binds = unique(binds)
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
	unlock, e := configLock()
	if e != nil {
		return e
	}
	defer unlock()
	if e := clearMissingPreferences(base, controllers()); e != nil {
		return e
	}
	c, e := loadConfig()
	if e != nil {
		return e
	}
	devices, binds, e := networkAccess(c)
	if e != nil {
		return e
	}
	var previous map[string]string
	readJSON(run+"/controller.json", &previous)
	selected := preferredController(controllers(), c.PreferredController, previous["serial"])
	if u, e := user.Lookup("homecontroller"); e == nil {
		gid, _ := strconv.Atoi(u.Gid)
		if usb := selected; usb != nil {
			bus, _ := strconv.Atoi(usb["bus"])
			dev, _ := strconv.Atoi(usb["device"])
			if bus > 0 && dev > 0 {
				p := fmt.Sprintf("/dev/bus/usb/%03d/%03d", bus, dev)
				os.Chown(p, 0, gid)
				os.Chmod(p, 0660)
			}
		}
	}
	for _, dir := range []string{base + "/state", run} {
		os.MkdirAll(dir, 0700)
		own(dir)
	}
	if e := atomicJSON(run+"/controller.json", selected); e != nil {
		return e
	}
	own(run + "/controller.json")
	if c.Timezone == "router" {
		b, err := exec.Command("/sbin/uci", "-q", "get", "system.@system[0].zonename").Output()
		if err != nil {
			return errors.New("router time zone unavailable")
		}
		zone := strings.TrimSpace(string(b))
		if _, err = time.LoadLocation(zone); err != nil {
			return err
		}
		if err = atomicJSON(run+"/timezone.json", zone); err != nil {
			return err
		}
		own(run + "/timezone.json")
	}
	if e := prepareIKEACredentials(); e != nil {
		return e
	}
	for _, p := range []string{"config.json", "credential.json"} {
		adminFile(base + "/" + p)
	}
	// Dedicated firewall table is evaluated before the regular router input chains.
	exec.Command("/usr/sbin/nft", "delete", "table", "inet", "home_controller").Run()
	quoted := []string{}
	for _, d := range devices {
		quoted = append(quoted, strconv.Quote(d))
	}
	rules := fmt.Sprintf("table inet home_controller {\n chain input {\n type filter hook input priority -10; policy accept;\n tcp dport %d iifname != { %s } drop;\n }\n}\n", c.Port, strings.Join(quoted, ", "))
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
	if e := writeRadioConfig(c, selected); e != nil {
		return e
	}
	if !exists(base+"/tls.crt") || !exists(base+"/tls.key") {
		if e := renewTLS(); e != nil {
			return e
		}
	}
	for _, p := range []string{base + "/tls.crt", base + "/tls.key"} {
		if !exists(p) {
			return errors.New("configure TLS certificate and key before enabling")
		}
		adminFile(p)
	}
	// Generate a shell argument file with validated addresses and fixed options only.
	lines := []string{}
	for _, a := range args {
		lines = append(lines, "'"+strings.ReplaceAll(a, "'", "'\\''")+"'")
	}
	p := filepath.Join(run, "uhttpd.args")
	f, err := os.CreateTemp(run, ".args-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, e = f.WriteString(strings.Join(lines, " "))
	f.Close()
	if e == nil {
		e = os.Rename(f.Name(), p)
	}
	own(p)
	return e
}
