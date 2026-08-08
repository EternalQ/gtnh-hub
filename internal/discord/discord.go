package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/EternalQ/gtnh-hub/internal/hub"
	"github.com/EternalQ/gtnh-hub/internal/util"
	"github.com/bwmarrin/discordgo"
)

type Config struct {
	BotToken     string            `yaml:"bot_token" env:"DISCORD_BOT_TOKEN"`
	WebhookID    string            `yaml:"webhook_id" env:"DISCORD_WEBHOOK_ID"`
	WebhookToken string            `yaml:"webhook_token" env:"DISCORD_WEBHOOK_TOKEN"`
	AdminRoleID  string            `yaml:"admin_role_id" env:"DISCORD_ADMIN_ROLE_ID"`
	PinnedMsg    string            `yaml:"pinned_msg" env:"DISCORD_PINNED_MSG"`
	PlayerAvaURL string            `yaml:"player_ava_url" env:"DISCORD_PLAYER_AVA_URL"`
	CommandRoles map[string]string `yaml:"command_roles"`
}

func (c Config) CommandRoleIDs(cmd string) []string {
	raw, ok := c.CommandRoles[cmd]
	if !ok {
		return nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			ids = append(ids, p)
		}
	}
	return ids
}

type Discord struct {
	Config
	WebhookChan string

	sess *discordgo.Session
}

func NewDiscord(cfg Config) (*Discord, error) {
	dg, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	wh, err := dg.Webhook(cfg.WebhookID)
	if err != nil {
		return nil, err
	}
	slog.Debug("Webhook created", slog.String("channel", wh.ChannelID))

	return &Discord{
		Config:      cfg,
		WebhookChan: wh.ChannelID,
		sess:        dg,
	}, nil
}

func (d *Discord) Setup(handlers ...any) error {
	for _, h := range handlers {
		d.sess.AddHandler(h)
	}
	return d.sess.Open()
}

func (d *Discord) UpdatePinned(count int, fields []*discordgo.MessageEmbedField) error {
	desc := fmt.Sprintf(
		"Общий онлайн игрков: %d.\nИнформация обновляется раз 10 сек.\nСлоты на серверах измеряются в командах (SU Teams).",
		count)
	msg := &discordgo.MessageEmbed{
		Title:       "Статус серверов",
		Description: desc,
		Fields:      fields,
	}
	_, err := d.sess.ChannelMessageEditEmbed(d.WebhookChan, d.PinnedMsg, msg)
	return err
}

func (d *Discord) SendChatMessage(source string, msg hub.ChatMessage) error {
	if msg.Sender == "" {
		_, err := d.sess.ChannelMessageSend(d.WebhookChan, msg.Text)
		return err
	}

	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(source)
	b.WriteString("] ")
	b.WriteString(util.CleanMinecraftTags(msg.SenderFormatted))
	sender := b.String()

	ava := d.PlayerAvaURL + msg.Sender

	p := &discordgo.WebhookParams{
		Content:   util.CleanMinecraftTags(msg.Text),
		Username:  sender,
		AvatarURL: ava,
	}

	_, err := d.sess.WebhookExecute(d.WebhookID, d.WebhookToken, false, p)
	if err != nil {
		return err
	}
	slog.Debug("Discord send",
		slog.String("Username", sender),
		slog.String("Content", util.CleanMinecraftTags(msg.Text)),
		slog.String("Avatar", ava))
	return nil
}

func (d *Discord) Close() {
	d.sess.Close()
	slog.Info("Discord closed")
}
