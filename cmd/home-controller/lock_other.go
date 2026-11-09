//go:build !linux

package main

import "sync"

// Development tests only. Production is built for Linux/OpenWrt.
var developmentLocks sync.Map

func lockFile(path string) (func(), error) {
	v, _ := developmentLocks.LoadOrStore(path, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return m.Unlock, nil
}
