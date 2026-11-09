package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func configLock() (func(), error) { return lockFile(base + "/.config.lock") }
func radioLock() (func(), error) {
	unlock, err := lockFile(run + "/radio.lock")
	if err == nil && os.Geteuid() == 0 {
		own(run + "/radio.lock")
	}
	return unlock, err
}

func connectedPreference(preferred string, available []map[string]string) string {
	for _, d := range available {
		if preferred != "" && d["serial"] == preferred {
			return preferred
		}
	}
	return ""
}

// A stale page must not re-save a preference for an unplugged/reinserted stick.
// USB identity is request metadata only, never a remembered device inventory.
func submittedPreference(preferred, instance string, available []map[string]string) string {
	for _, d := range available {
		if d["serial"] == preferred && instance != "" && instance == d["bus"]+":"+d["device"] {
			return preferred
		}
	}
	return ""
}
func preferredController(available []map[string]string, preferred, previous string) map[string]string {
	if preferred = connectedPreference(preferred, available); preferred != "" {
		return selectController(available, preferred)
	}
	return selectController(available, previous)
}
func sameController(a, b map[string]string) bool {
	return a["serial"] == b["serial"] && a["bus"] == b["bus"] && a["device"] == b["device"]
}

// Forget missing preferences in both active and staged settings, preserving
// all other edits. There is no persistent inventory of disconnected devices.
// Callers serialize this with Save/Apply through the configuration lock.
func clearMissingPreferences(dir string, available []map[string]string) error {
	for _, name := range []string{"config.json", "pending.json"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) && name == "pending.json" {
			continue
		}
		if err != nil {
			return err
		}
		c, err := parseConfig(strings.NewReader(string(data)))
		if err != nil {
			return err
		}
		if connectedPreference(c.PreferredController, available) == c.PreferredController {
			continue
		}
		c.PreferredController = ""
		if err := atomicJSON(path, c); err != nil {
			return err
		}
		if dir == base && name == "config.json" {
			adminFile(path)
		}
	}
	return nil
}

func writeRadioConfig(c Config, selected map[string]string) error {
	var radio strings.Builder
	radio.WriteString("# Managed by Home Controller\nuser = \"homecontroller\"\ngroup = \"homecontroller\"\n")
	if selected != nil {
		fmt.Fprintf(&radio, "controller {\n id = 1\n type = 1\n serial = %q\n}\n", selected["serial"])
	}
	for _, d := range c.Devices {
		fmt.Fprintf(&radio, "device {\n id = %d\n name = %q\n protocol = \"hasta\"\n model = \"selflearningv2\"\n controller = 1\n parameters {\n house = %d\n unit = %d\n }\n}\n", d.RadioID, d.ID, d.House, d.Unit)
	}
	if err := os.WriteFile("/etc/tellstick.conf", []byte(radio.String()), 0600); err != nil {
		return err
	}
	adminFile("/etc/tellstick.conf")
	return nil
}

func refreshRadio(c Config) error {
	if !c.Enabled {
		return nil
	}
	unlock, err := radioLock()
	if err != nil {
		return err
	}
	defer unlock()
	available := controllers()
	if err := clearMissingPreferences(base, available); err != nil {
		return err
	}
	var previous map[string]string
	readJSON(run+"/controller.json", &previous)
	selected := preferredController(available, c.PreferredController, previous["serial"])
	if sameController(previous, selected) {
		return nil
	}
	if selected != nil {
		bus, _ := strconv.Atoi(selected["bus"])
		dev, _ := strconv.Atoi(selected["device"])
		if bus < 1 || dev < 1 {
			return errors.New("invalid TellStick USB address")
		}
		path := fmt.Sprintf("/dev/bus/usb/%03d/%03d", bus, dev)
		// Reuse the existing root:homecontroller USB permissions policy.
		adminFile(path)
		if err := os.Chmod(path, 0660); err != nil {
			return err
		}
	}
	// The driver is unavailable during replacement; never report stale hardware
	// as active, and never replay a command that was interrupted by unplugging.
	if err := atomicJSON(run+"/controller.json", nil); err != nil {
		return err
	}
	own(run + "/controller.json")
	if err := writeRadioConfig(c, selected); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "/etc/init.d/telldusd", "restart").Run(); err != nil {
		return errors.New("could not restart the TellStick driver")
	}
	if err := atomicJSON(run+"/controller.json", selected); err != nil {
		return err
	}
	own(run + "/controller.json")
	return nil
}

func usbChange() error {
	if os.Geteuid() != 0 {
		return errors.New("administrator required")
	}
	unlock, err := configLock()
	if err != nil {
		return err
	}
	defer unlock()
	available := controllers()
	// The removal path also catches an active stick unplugged and reinserted
	// before this hook is scheduled. Do not restore its former preference.
	var previous map[string]string
	readJSON(run+"/controller.json", &previous)
	if os.Getenv("ACTION") == "remove" && previous["path"] != "" && os.Getenv("DEVPATH") == previous["path"] {
		filtered := []map[string]string{}
		for _, d := range available {
			if d["serial"] != previous["serial"] {
				filtered = append(filtered, d)
			}
		}
		available = filtered
	}
	if err := clearMissingPreferences(base, available); err != nil {
		return err
	}
	time.Sleep(750 * time.Millisecond)
	if err := clearMissingPreferences(base, controllers()); err != nil {
		return err
	}
	c, err := loadConfig()
	if err != nil {
		return err
	}
	return refreshRadio(c)
}
