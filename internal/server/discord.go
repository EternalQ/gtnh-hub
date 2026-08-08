package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/EternalQ/gtnh-hub/internal/game"
	"github.com/EternalQ/gtnh-hub/internal/hub"
	"github.com/EternalQ/gtnh-hub/internal/util"
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

func (s *Server) handleCommand(sess *discordgo.Session, m *discordgo.MessageCreate) {
	fields := strings.Fields(m.Content)
	if len(fields) == 0 {
		return
	}
	cmd := strings.TrimPrefix(fields[0], "!")
	args := fields[1:]

	slog.Info("Recieved discord command", slog.String("cmd", cmd), slog.String("caller", m.Author.Username))

	if !s.hasCommandPermission(sess, m, cmd) {
		if _, err := sess.ChannelMessageSendReply(s.ds.WebhookChan, "Недостаточно прав", m.Reference()); err != nil {
			slog.Error("Discord commands reject", slog.String("err", err.Error()))
		}
		return
	}

	switch cmd {
	case "ping":
		sess.ChannelMessageSendReply(m.ChannelID, "Pong! 🏓", m.Reference())
	case "check":
		s.handleCheckCommand(sess, m, args)
	default:
		sess.ChannelMessageSendReply(m.ChannelID, "Команда не найдена: "+m.Content, m.MessageReference)
	}
}

func (s *Server) hasCommandPermission(sess *discordgo.Session, m *discordgo.MessageCreate, cmd string) bool {
	if s.ds.AdminRoleID != "" && slices.Contains(m.Member.Roles, s.ds.AdminRoleID) {
		return true
	}

	allowedIDs := s.ds.CommandRoleIDs(cmd)
	if len(allowedIDs) == 0 {
		return false
	}

	for _, rid := range m.Member.Roles {
		if slices.Contains(allowedIDs, rid) {
			return true
		}
	}

	roles, err := sess.GuildRoles(m.GuildID)
	if err != nil {
		slog.Error("Discord fetch guild roles", slog.String("err", err.Error()))
		return false
	}

	positions := make(map[string]int, len(roles))
	for _, r := range roles {
		positions[r.ID] = r.Position
	}

	maxAllowedPos := -1
	for _, rid := range allowedIDs {
		if pos, ok := positions[rid]; ok && pos > maxAllowedPos {
			maxAllowedPos = pos
		}
	}

	for _, rid := range m.Member.Roles {
		if pos, ok := positions[rid]; ok && pos > maxAllowedPos {
			return true
		}
	}

	return false
}

func (s *Server) handleCheckCommand(sess *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) != 1 {
		sess.ChannelMessageSendReply(m.ChannelID, "Использование: !check <server>", m.Reference())
		return
	}

	tag := args[0]
	instanceID, daemonID, ok := s.mcs.ResolveInstance(tag)
	if !ok {
		sess.ChannelMessageSendReply(m.ChannelID, "Неизвестный сервер: "+tag, m.Reference())
		return
	}

	result, err := s.checkGameInstance(tag, instanceID, daemonID)
	if err != nil {
		slog.Error("Discord check command", slog.String("tag", tag), slog.String("err", err.Error()))
		sess.ChannelMessageSendReply(m.ChannelID, "Ошибка проверки "+tag+": "+err.Error(), m.Reference())
		return
	}

	statusName := game.StatusName(result.PanelStatus)

	var reply string
	switch {
	case result.RConOk:
		reply = fmt.Sprintf("%s: статус в панели — %s, RCon отвечает за %s", tag, statusName, result.RConLatency)
	case result.Killed:
		reply = fmt.Sprintf("%s: статус в панели — %s, но RCon недоступен. Сервер убит (kill), ожидается перезапуск", tag, statusName)
	default:
		reply = fmt.Sprintf("%s: статус в панели — %s", tag, statusName)
	}

	sess.ChannelMessageSendReply(m.ChannelID, reply, m.Reference())
}

func (s *Server) dsPinMsgUpdater() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fields := make([]*discordgo.MessageEmbedField, 0)
		for id, gs := range s.game.Snapshot() {
			stat := gs.Stat()
			if stat == nil {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:  fmt.Sprintf("%s - выключен", id),
					Value: "Здесь пусто(",
				})
				continue
			}

			if len(stat.OnlinePlayers) == 0 {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:  fmt.Sprintf("%s (%.2f TPS) - 0/%d", id, stat.Tps, stat.Slots),
					Value: "Здесь пусто(",
				})
				continue
			}

			list := fmt.Sprintf("**%d игрок(-а, -ов)**:	", len(stat.OnlinePlayers))
			teams := make(map[string]struct{})
			for _, player := range stat.OnlinePlayers {
				teams[player.Team] = struct{}{}
				list = fmt.Sprintf("%s\n%s - %s", list, util.CleanMinecraftTags(player.NameFormatted), player.Team)
			}

			fields = append(fields, &discordgo.MessageEmbedField{
				Name:  fmt.Sprintf("%s (%.2f TPS) - %d/%d", id, stat.Tps, len(teams), stat.Slots),
				Value: list,
			})
		}

		if err := s.ds.UpdatePinned(s.game.AllPlayersCount(), fields); err != nil {
			slog.Error("Discord command error", slog.String("err", err.Error()))
		}
	}
}
