package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

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
				if err := s.ds.SendChatMessage(msg.Origin, p); err != nil {
					slog.Error("Discord send ", slog.String("err", err.Error()))
				}
			case hub.ActionPlayerLogged:
				// ignore
			default:
				slog.Warn("Discord message handler: Unimplemented action", slog.String("action", msg.Action))
			}
		}
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

func (s *Server) onlineRefresher(sess *discordgo.Session, chanId, msgId string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		inst, err := s.game.Copy()
		if err != nil {
			// TODO: notify
			slog.Error("Discord command handler", slog.String("err", err.Error()))
			return
		}
		fields := make([]*discordgo.MessageEmbedField, 0)
		for id, server := range inst.GameServers {
			if server == nil {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:  fmt.Sprintf("%s - выключен", id),
					Value: "Здесь пусто(",
				})
				continue
			}

			if len(server.OnlinePlayers) == 0 {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:  fmt.Sprintf("%s (%.2f TPS) - 0/%d", id, server.Tps, server.Slots),
					Value: "Здесь пусто(",
				})
				continue
			}

			list := fmt.Sprintf("%d игрок(-а, -ов):\n", len(server.OnlinePlayers))
			teams := make(map[string]struct{})
			for _, player := range server.OnlinePlayers {
				teams[player.Team] = struct{}{}
				list = fmt.Sprintf("%s\n%s - %s", list, player.NameFormatted, player.Team)
			}

			fields = append(fields, &discordgo.MessageEmbedField{
				Name:  fmt.Sprintf("%s (%.2f TPS) - %d/%d", id, server.Tps, len(teams), server.Slots),
				Value: list,
			})
		}

		// TODO: take from config
		desc := fmt.Sprintf("Общий онлайн игрков: %d.\nИнформация обновляется раз 10 сек.\nСлоты на серверах измеряются в командах (SU Teams).",
			s.game.GetAllPlayersCount())
		msg := &discordgo.MessageEmbed{
			Title:       "Статус серверов",
			Description: desc,
			Fields:      fields,
		}
		if _, err := sess.ChannelMessageEditEmbed(chanId, msgId, msg); err != nil {
			slog.Error("Discord command error", slog.String("err", err.Error()))
		}
	}
}

func (s *Server) handleCommand(sess *discordgo.Session, m *discordgo.MessageCreate) {
	slog.Info("Recieved discord command", slog.String("cmd", m.Content), slog.String("caller", m.Author.Username))
	if !slices.Contains(m.Member.Roles, s.ds.AdminRoleId) {
		_, err := sess.ChannelMessageSendReply(s.ds.WebhookChan, "Недостаточно прав", m.Reference())
		if err != nil {
			slog.Error("Discord commands reject", slog.String("err", err.Error()))
		}
		return
	}

	switch m.Content {
	case "!ping":
		sess.ChannelMessageSendReply(m.ChannelID, "Pong! 🏓", m.Reference())

	default:
		sess.ChannelMessageSendReply(m.ChannelID, "Команда не найдена: "+m.Content, m.MessageReference)
	}
}
