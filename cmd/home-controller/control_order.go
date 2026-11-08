package main

import "errors"

// Typed keys let a device named "scheduler" coexist with the scheduler control.
func defaultControlKeys(c Config) []string {
	keys := make([]string, 0, len(c.Devices)+len(c.Groups)+1)
	for _, d := range c.Devices {
		keys = append(keys, "device:"+d.ID)
	}
	keys = append(keys, "scheduler")
	for _, g := range c.Groups {
		keys = append(keys, "group:"+g.ID)
	}
	return keys
}

func validateControlOrder(c Config) error {
	allowed := map[string]bool{}
	for _, key := range defaultControlKeys(c) {
		allowed[key] = true
	}
	seen := map[string]bool{}
	for _, key := range c.ControlOrder {
		if !allowed[key] || seen[key] {
			return errors.New("control order contains an unknown or repeated control")
		}
		seen[key] = true
	}
	return nil
}

func orderedControlKeys(c Config) []string {
	defaults := defaultControlKeys(c)
	allowed, seen := map[string]bool{}, map[string]bool{}
	for _, key := range defaults {
		allowed[key] = true
	}
	keys := make([]string, 0, len(defaults))
	for _, key := range append(append([]string{}, c.ControlOrder...), defaults...) {
		if allowed[key] && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	return keys
}
