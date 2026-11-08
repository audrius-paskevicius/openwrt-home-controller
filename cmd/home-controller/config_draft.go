package main

import (
	"errors"
	"os"
	"path/filepath"
)

// Drafts are root-only and never read by the running service or at startup.
func editableConfig(dir string, active Config) (Config, bool, error) {
	f, err := os.Open(filepath.Join(dir, "pending.json"))
	if errors.Is(err, os.ErrNotExist) {
		return active, false, nil
	}
	if err != nil {
		return Config{}, false, err
	}
	defer f.Close()
	c, err := parseConfig(f)
	return c, true, err
}

func discardConfig(dir string) error {
	err := os.Remove(filepath.Join(dir, "pending.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func storeConfig(dir string, c Config, activate bool) error {
	if err := validate(c); err != nil {
		return err
	}
	if !activate {
		return atomicJSON(filepath.Join(dir, "pending.json"), c)
	}
	if err := atomicJSON(filepath.Join(dir, "config.json"), c); err != nil {
		return err
	}
	return discardConfig(dir)
}
