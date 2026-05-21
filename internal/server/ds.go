package server

import (
	"log/slog"
	"strings"

	"github.com/EternalQ/gtnh-hub/internal/chat"
	"github.com/bwmarrin/discordgo"
)

func (s *Server) dsConnect(sess *discordgo.Session, m *discordgo.Connect) {
	ch := make(chan chat.Message, 100)

	go func() {
		for msg := range ch {
			if err := s.ds.Send(msg); err != nil {
				slog.Error("Discord send ", slog.String("err", err.Error()))
			}
		}
		slog.Info("Channel closed: Discord")
	}()

	s.chat.Register("Discord", ch)
}

func (s *Server) dsHandler(sess *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == sess.State.User.ID ||
		m.ChannelID != s.ds.WebhookChan ||
		m.WebhookID != "" {
		return
	}
	if strings.HasPrefix(m.Content, "!ping") {
		sess.ChannelMessageSend(m.ChannelID, "Pong! 🏓")
		return
	}

	// TODO: check role and add [A] for admins

	s.chat.Send(chat.Message{
		Server:          "Discord",
		Sender:          m.Author.GlobalName,
		SenderFormatted: m.Author.GlobalName,
		Text:            m.Content,
	})
}

func (s *Server) dsDisconnect(sess *discordgo.Session, m *discordgo.Disconnect) {
	s.chat.Unregister("Discord")
}
