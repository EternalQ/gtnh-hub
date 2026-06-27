package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/EternalQ/gtnh-hub/internal/hub"
	"github.com/bwmarrin/discordgo"
)

func (s *Server) dsConnect(sess *discordgo.Session, m *discordgo.Connect) {
	ch := make(chan hub.Message, 100)

	go func() {
		for msg := range ch {
			switch msg.Action {
			case hub.ActionChat:
				var p hub.ChatMessage
				if err := json.Unmarshal(msg.Payload, &p); err != nil {
					slog.Error("failed to unmarshal chat message payload ", slog.String("err", err.Error()))
				}
				if err := s.ds.Send(msg.Origin, p); err != nil {
					slog.Error("Discord send ", slog.String("err", err.Error()))
				}
			case hub.ActionPlayerLogged:
				// ignore
			default:
				slog.Warn("Discord message handler: Unimplemented action", slog.String("action", msg.Action))
			}
		}
		slog.Info("Channel closed: Discord")
	}()

	s.hub.Register("Discord", ch)
}

func (s *Server) dsDisconnect(sess *discordgo.Session, m *discordgo.Disconnect) {
	s.hub.Unregister("Discord")
}

func (s *Server) dsHandler(sess *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == sess.State.User.ID ||
		m.ChannelID != s.ds.WebhookChan ||
		m.WebhookID != "" {
		return
	}
	if strings.HasPrefix(m.Content, "!") {
		s.handleCommand(sess, m)
		return
	}

	// TODO: check role and add [A] for admins

	s.hub.SendRaw("Discord", hub.ActionChat, &hub.ChatMessage{
		Sender:          m.Author.GlobalName,
		SenderFormatted: m.Author.GlobalName,
		Text:            m.Content,
	})
}

func (s *Server) handleCommand(sess *discordgo.Session, m *discordgo.MessageCreate) {
	switch m.Content {
	case "!ping":
		sess.ChannelMessageSend(m.ChannelID, "Pong! 🏓")
	case "!online":
		fields := make([]*discordgo.MessageEmbedField, len(s.game.GameServers))
		var b strings.Builder
		for id, server := range s.game.GameServers {
			teams := make(map[string]struct{})
			for _, player := range server.OnlinePlayers {
				teams[player.Team] = struct{}{}
				b.WriteString(player.Name)
				b.WriteString(" — ")
				b.WriteString(player.Team)
				b.WriteString("\n")
			}

			fields = append(fields, &discordgo.MessageEmbedField{
				Name:  fmt.Sprintf("%s - %d/%d", id, len(teams), server.Slots),
				Value: b.String(),
			})

			b.Reset()
		}

		desc := fmt.Sprintf("Общий онлайн игрков: %d.\nСлоты на серверах изменяются в командах", s.game.GetAllPlayersCount())
		msg := &discordgo.MessageEmbed{
			Title:       "Статус серверов",
			Description: desc,
			Fields:      fields,
		}
		sess.ChannelMessageSendEmbed(m.ChannelID, msg)
	}
}
