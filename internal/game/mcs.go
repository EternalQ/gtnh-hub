package game

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	StatusBusy     = -1
	StatusStopped  = 0
	StatusStopping = 1
	StatusStarting = 2
	StatusRunning  = 3
)

func StatusName(status int) string {
	switch status {
	case StatusBusy:
		return "Busy"
	case StatusStopped:
		return "Stopped"
	case StatusStopping:
		return "Stopping"
	case StatusStarting:
		return "Starting"
	case StatusRunning:
		return "Running"
	default:
		return fmt.Sprintf("Unknown(%d)", status)
	}
}

type MCSClient struct {
	GameConfig

	httpClient *http.Client
}

func NewMCSClient(cfg GameConfig) *MCSClient {
	return &MCSClient{
		GameConfig: cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type mcsEnvelope[T any] struct {
	Status int   `json:"status"`
	Data   T     `json:"data"`
	Time   int64 `json:"time"`
}

type mcsInstanceDetail struct {
	Status int `json:"status"`
}

func (c *MCSClient) InstanceStatus(ctx context.Context, daemonID, uuid string) (int, error) {
	var env mcsEnvelope[mcsInstanceDetail]
	if err := c.do(ctx, "/api/instance", url.Values{"uuid": {uuid}, "daemonId": {daemonID}}, &env); err != nil {
		return 0, err
	}
	if env.Status != http.StatusOK {
		return 0, fmt.Errorf("mcs api: instance status request failed, code %d", env.Status)
	}
	return env.Data.Status, nil
}

func (c *MCSClient) StartInstance(ctx context.Context, daemonID, uuid string) error {
	var env mcsEnvelope[map[string]any]
	if err := c.do(ctx, "/api/protected_instance/open", url.Values{"uuid": {uuid}, "daemonId": {daemonID}}, &env); err != nil {
		return err
	}
	if env.Status != http.StatusOK {
		return fmt.Errorf("mcs api: open instance failed, code %d", env.Status)
	}
	return nil
}

func (c *MCSClient) RestartInstance(ctx context.Context, daemonID, uuid string) error {
	var env mcsEnvelope[map[string]any]
	if err := c.do(ctx, "/api/protected_instance/restart", url.Values{"uuid": {uuid}, "daemonId": {daemonID}}, &env); err != nil {
		return err
	}
	if env.Status != http.StatusOK {
		return fmt.Errorf("mcs api: restart instance failed, code %d", env.Status)
	}
	return nil
}

func (c *MCSClient) KillInstance(ctx context.Context, daemonID, uuid string) error {
	var env mcsEnvelope[map[string]any]
	if err := c.do(ctx, "/api/protected_instance/kill", url.Values{"uuid": {uuid}, "daemonId": {daemonID}}, &env); err != nil {
		return err
	}
	if env.Status != http.StatusOK {
		return fmt.Errorf("mcs api: kill instance failed, code %d", env.Status)
	}
	return nil
}

func (c *MCSClient) OutputLog(ctx context.Context, daemonID, uuid string, sizeKB int) (string, error) {
	q := url.Values{"uuid": {uuid}, "daemonId": {daemonID}}
	if sizeKB > 0 {
		q.Set("size", strconv.Itoa(sizeKB))
	}

	var env mcsEnvelope[string]
	if err := c.do(ctx, "/api/protected_instance/outputlog", q, &env); err != nil {
		return "", err
	}
	if env.Status != http.StatusOK {
		return "", fmt.Errorf("mcs api: output log request failed, code %d", env.Status)
	}
	return env.Data, nil
}

func (c *MCSClient) do(ctx context.Context, path string, query url.Values, out any) error {
	query.Set("apikey", c.MCSAPIKey)
	u := strings.TrimRight(c.MCSBaseURL, "/") + path + "?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mcs api: unexpected http status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
