package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type CheckinRequest struct {
	AgentVersion  string        `json:"agentVersion"`
	SentAt        time.Time     `json:"sentAt"`
	UptimeSec     int64         `json:"uptimeSec"`
	ConfigVersion int           `json:"configVersion"`
	Reason        string        `json:"reason"`
	BeaconList    []BeaconState `json:"beaconList"`
}

type CheckinClient struct {
	http *http.Client
}

func NewCheckinClient() *CheckinClient {
	return &CheckinClient{http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *CheckinClient) Send(ctx context.Context, cfg Config, req CheckinRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	url := strings.TrimRight(cfg.ServerURL, "/") + "/deviceCheckin"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.DeviceToken)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("check-in: HTTP %d: %s", resp.StatusCode, bytes.TrimSpace(snippet))
	}
	return nil
}
