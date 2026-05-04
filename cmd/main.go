package main

import (
	"context"
	"fmt"
	"log"
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
	port         string
	dsBotToken   string
	dsWhId       string
	dsWhToken    string
	playerAvaUrl string
)

func init() {
	viper.SetDefault("PORT", "5665")

	viper.AutomaticEnv()

	port = viper.GetString("PORT")
	dsBotToken = viper.GetString("DS_BOT_TOKEN")
	dsWhId = viper.GetString("DS_WH_ID")
	dsWhToken = viper.GetString("DS_WH_TOKEN")
	playerAvaUrl = viper.GetString("PLAYER_AVA_URL")

	if len(dsBotToken) == 0 || len(dsWhId) == 0 {
		log.Fatal("check discord credentials")
	}
}

func main() {
	ds, err := discord.NewDiscord(dsBotToken, dsWhId, dsWhToken, playerAvaUrl)
	if err != nil {
		fmt.Printf("Discord creation error: %s\n", err.Error())
	}

	hub := chat.NewHub()
	srv := server.NewServer(ds, hub)

	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("Server started on %s\n", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Graceful shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("Server Shutdown Failed: %s", err.Error())
	}
	hub.Close()
	ds.Close()
}
