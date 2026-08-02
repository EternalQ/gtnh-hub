package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EternalQ/gtnh-hub/internal/config"
	"github.com/EternalQ/gtnh-hub/internal/discord"
	"github.com/EternalQ/gtnh-hub/internal/hub"
	"github.com/EternalQ/gtnh-hub/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config: %s", err.Error())
	}

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logH := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.Debug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time().In(time.FixedZone("GMT+3", 3*60*60))
				if cfg.Debug {
					return slog.Time(slog.TimeKey, t)
				}
				return slog.String(slog.TimeKey, t.Format("2006-01-02 15:04:05"))
			}
			return a
		},
	})
	slog.SetDefault(slog.New(logH))

	ds, err := discord.NewDiscord(cfg.Discord)
	if err != nil {
		slog.Error("Discord creation", slog.String("err", err.Error()))
	}

	hub := hub.NewHub()
	srv, err := server.NewServer(ds, hub)
	if err != nil {
		log.Fatalf("Server creation: %s", err.Error())
	}

	httpSrv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("Server started", slog.String("port", cfg.Port))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("Graceful shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		slog.Error("Server Shutdown", slog.String("err", err.Error()))
	}
	hub.Close()
	ds.Close()
}
