//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRadioLockSerializesAndRejectsSymlinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock")
	unlock, err := lockFile(path)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		release, e := lockFile(path)
		if e == nil {
			release()
		}
		done <- e
	}()
	select {
	case <-done:
		unlock()
		t.Fatal("second operation bypassed lock")
	case <-time.After(75 * time.Millisecond):
	}
	unlock()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lock was not released")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if release, e := lockFile(link); e == nil {
		release()
		t.Fatal("followed a lock symlink")
	}
}
