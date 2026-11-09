package main

import "errors"

func (d Device) deviceType() string {
	if d.Type == "" {
		return "blind"
	} // Existing configurations remain blinds.
	return d.Type
}

func validateDeviceType(d Device) error {
	if d.deviceType() != "ikea" && d.IKEA != nil {
		return errors.New("IKEA target requires IKEA device type")
	}
	switch d.deviceType() {
	case "ikea":
		if d.IKEA == nil || d.IKEA.Gateway == "" || len(d.IKEA.Gateway) > 80 || d.IKEA.ID < 1 || d.IKEA.ID > 2147483647 || (d.IKEA.Kind != "device" && d.IKEA.Kind != "group") || !d.IKEA.Capabilities.Power || d.RadioID != 0 || d.Protocol != "" || d.HouseCode != "" || d.House != 0 || d.Unit != 0 {
			return errors.New("choose an IKEA device or group")
		}
	case "blind":
		if (d.Protocol != "" && d.Protocol != "hasta") || d.HouseCode != "" || d.House < 1 || d.House > 65535 || d.Unit < 0 || d.Unit > 15 {
			return errors.New("blind requires Hasta, house 1..65535 and unit 0..15")
		}
	case "switch", "dimmer":
		if (d.Protocol != "" && d.Protocol != "arctech") || d.House != 0 || len(d.HouseCode) != 1 || d.HouseCode[0] < 'A' || d.HouseCode[0] > 'P' || d.Unit < 1 || d.Unit > 16 {
			return errors.New("switch or BH004 dimmer requires Arctech code switch, house A..P and unit 1..16")
		}
	default:
		return errors.New("unsupported device type")
	}
	return nil
}

func (c Config) targetType(target string) string {
	for _, d := range c.Devices {
		if d.ID == target {
			return d.deviceType()
		}
	}
	for _, g := range c.Groups {
		if g.ID != target {
			continue
		}
		kind := ""
		for _, id := range g.Devices {
			found := false
			for _, d := range c.Devices {
				if d.ID != id {
					continue
				}
				if kind != "" && kind != d.deviceType() {
					return ""
				}
				kind, found = d.deviceType(), true
			}
			if !found {
				return ""
			}
		}
		return kind
	}
	return ""
}

func (c Config) supportsAction(target, action string) bool {
	if c.targetType(target) == "ikea" {
		for _, d := range c.Devices {
			if d.ID == target && d.IKEA != nil {
				return d.IKEA.Capabilities.supports(action)
			}
		}
		for _, g := range c.Groups {
			if g.ID == target {
				for _, id := range g.Devices {
					if !c.supportsAction(id, action) {
						return false
					}
				}
				return true
			}
		}
		return false
	}
	switch c.targetType(target) {
	case "blind":
		return action == "up" || action == "stop" || action == "down"
	case "switch":
		return action == "on" || action == "off"
	case "dimmer":
		return action == "toggle" || action == "dim"
	}
	return false
}

// BH004 has two state-dependent commands, not absolute power or brightness.
func (d Device) radioAction(action string) string {
	if d.deviceType() == "dimmer" {
		if action == "toggle" {
			return "off"
		}
		if action == "dim" {
			return "on"
		}
	}
	return action
}
