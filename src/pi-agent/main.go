// Command pi-agent runs on the in-home Raspberry Pi: it scans for the
// configured BLE bands and checks in with the server on a schedule, and at
// once when a band comes or goes.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

var agentVersion = "dev" // set by -ldflags at build

func main() {
	configPath := flag.String("config", "/etc/stillthere/agent.json", "agent config file")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tracker := NewTracker(cfg.BeaconList)
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		log.Error("bluetooth enable", "err", err)
		os.Exit(1)
	}
	scanErr := make(chan error, 1)
	go func() {
		scanErr <- adapter.Scan(func(_ *bluetooth.Adapter, r bluetooth.ScanResult) {
			tracker.Observe(r.Address.String(), int(r.RSSI), time.Now())
		})
	}()

	client := NewCheckinClient()
	started := time.Now()
	checkin := func(reason string, stateList []BeaconState) {
		req := CheckinRequest{
			AgentVersion:  agentVersion,
			SentAt:        time.Now().UTC(),
			UptimeSec:     int64(time.Since(started).Seconds()),
			ConfigVersion: cfg.ConfigVersion,
			Reason:        reason,
			BeaconList:    stateList,
		}
		for _, b := range stateList {
			log.Info("beacon", "reason", reason, "address", b.Address, "label", b.Label, "present", b.Present, "rssi", derefInt(b.RSSI))
		}
		if cfg.ServerURL == "" {
			return
		}
		sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if err := client.Send(sendCtx, cfg, req); err != nil {
			log.Warn("check-in failed", "reason", reason, "err", err)
			return
		}
		log.Info("checked in", "reason", reason)
	}

	// Presence is re-evaluated every second so a change is reported within
	// a second of crossing absentAfter, not at the next scheduled check-in.
	presentMap := map[string]bool{}
	nextCheckin := time.Now()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	log.Info("started", "version", agentVersion, "beacons", len(cfg.BeaconList), "interval", cfg.CheckinInterval())
	for {
		select {
		case <-ctx.Done():
			adapter.StopScan()
			log.Info("stopped")
			return
		case err := <-scanErr:
			log.Error("scan ended", "err", err)
			os.Exit(1) // systemd restarts the agent
		case now := <-tick.C:
			stateList := tracker.Snapshot(now, cfg.AbsentAfter())
			reason := ""
			for _, b := range stateList {
				if was, seen := presentMap[b.Address]; !seen || was != b.Present {
					presentMap[b.Address] = b.Present
					if seen {
						reason = "presence-change"
					}
				}
			}
			if reason == "" && !now.Before(nextCheckin) {
				reason = "scheduled"
			}
			if reason != "" {
				checkin(reason, stateList)
				nextCheckin = now.Add(cfg.CheckinInterval())
			}
		}
	}
}

func derefInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
