package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config is the agent's saved state. The server owns every field except
// ServerURL and DeviceToken, which are set at install; a later check-in
// reply replaces the rest and bumps ConfigVersion.
type Config struct {
	ServerURL          string   `json:"serverUrl"`
	DeviceToken        string   `json:"deviceToken"`
	ConfigVersion      int      `json:"configVersion"`
	CheckinIntervalSec int      `json:"checkinIntervalSec"`
	AbsentAfterSec     int      `json:"absentAfterSec"`
	BeaconList         []Beacon `json:"beaconList"`
}

type Beacon struct {
	Address string `json:"address"`
	Label   string `json:"label"`
}

func (c Config) CheckinInterval() time.Duration {
	return time.Duration(c.CheckinIntervalSec) * time.Second
}

func (c Config) AbsentAfter() time.Duration {
	return time.Duration(c.AbsentAfterSec) * time.Second
}

func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	for i := range cfg.BeaconList {
		cfg.BeaconList[i].Address = strings.ToUpper(cfg.BeaconList[i].Address)
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	var errList []error
	if c.CheckinIntervalSec < 10 {
		errList = append(errList, errors.New("checkinIntervalSec must be at least 10"))
	}
	if c.AbsentAfterSec < 10 {
		errList = append(errList, errors.New("absentAfterSec must be at least 10"))
	}
	if len(c.BeaconList) == 0 {
		errList = append(errList, errors.New("beaconList is empty"))
	}
	return errors.Join(errList...)
}

// SaveConfig writes through a temp file and rename, so a power cut mid-write
// leaves the old file, never a torn one.
func SaveConfig(path string, cfg Config) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agent-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
