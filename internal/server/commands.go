package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/EternalQ/gtnh-hub/internal/game"
)

type CheckResult struct {
	PanelStatus int
	RConOk      bool
	RConLatency time.Duration
	Killed      bool
}

func (s *Server) checkGameInstance(tag, instanceID, daemonID string) (CheckResult, error) {
	var result CheckResult

	ctx := context.Background()

	status, err := s.mcs.InstanceStatus(ctx, daemonID, instanceID)
	if err != nil {
		return result, fmt.Errorf("panel status: %w", err)
	}
	result.PanelStatus = status

	if status != game.StatusRunning {
		return result, nil
	}

	addr, ok := s.game.RConAddr(tag)
	if !ok {
		return result, fmt.Errorf("no rcon address known for instance %q", tag)
	}

	start := time.Now()
	err = game.PingRCon(addr, s.mcs.RConPassword)
	for attempts := 0; err != nil && attempts < 3; attempts++ {
		time.Sleep(1 * time.Second)
		err = game.PingRCon(addr, s.mcs.RConPassword)
	}
	if err == nil {
		result.RConOk = true
		result.RConLatency = time.Since(start)
		return result, nil
	}

	slog.Warn("Game instance running but unreachable via RCon, killing",
		slog.String("instance", tag),
		slog.String("err", err.Error()),
	)

	result.Killed = true
	if err := s.mcs.KillInstance(ctx, daemonID, instanceID); err != nil {
		return result, fmt.Errorf("kill instance: %w", err)
	}
	if err := s.mcs.StartInstance(ctx, daemonID, instanceID); err != nil {
		return result, fmt.Errorf("start instance: %w", err)
	}
	return result, nil
}
