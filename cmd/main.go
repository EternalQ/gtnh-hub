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

	"github.com/EternalQ/gtnh-hub/internal/chat"
	"github.com/EternalQ/gtnh-hub/internal/discord"
	"github.com/EternalQ/gtnh-hub/internal/server"
	"github.com/spf13/viper"
)

var (
	port  string
	debug bool

	dsBotToken   string
	dsWhId       string
	dsWhToken    string
	playerAvaUrl string
)

func init() {
	viper.SetDefault("PORT", "5665")
	viper.SetDefault("DEBUG", false)

	viper.AutomaticEnv()

	port = viper.GetString("PORT")
	debug = viper.GetBool("DEBUG")

	dsBotToken = viper.GetString("DS_BOT_TOKEN")
	dsWhId = viper.GetString("DS_WH_ID")
	dsWhToken = viper.GetString("DS_WH_TOKEN")
	playerAvaUrl = viper.GetString("PLAYER_AVA_URL")

	if len(dsBotToken) == 0 || len(dsWhId) == 0 {
		log.Fatal("check discord credentials")
	}
}

func main() {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	logH := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: debug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time().In(time.FixedZone("GMT+3", 3*60*60))
				if debug {
					return slog.Time(slog.TimeKey, t)
				}
				return slog.String(slog.TimeKey, t.Format("2006-01-02 15:04:05"))
			}
			return a
		},
	})
	slog.SetDefault(slog.New(logH))

	ds, err := discord.NewDiscord(dsBotToken, dsWhId, dsWhToken, playerAvaUrl)
	if err != nil {
		slog.Error("Discord creation", slog.String("err", err.Error()))
	}

	hub := chat.NewHub()
	srv, err := server.NewServer(ds, hub)
	if err != nil {
		log.Fatalf("Server creation: %s", err.Error())
	}

	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("Server started", slog.String("port", port))
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
