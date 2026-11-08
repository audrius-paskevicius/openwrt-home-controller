package main

// Symbols are presentation only. No user-supplied markup or resource URLs.
func validIcon(icon string) bool {
	switch icon {
	case "", "none", "blind", "clock", "lamp", "plug", "fan":
		return true
	}
	return false
}
