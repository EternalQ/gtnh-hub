package discord

import (
	"log/slog"
	"strings"

	"github.com/EternalQ/gtnh-hub/internal/chat"
	"github.com/EternalQ/gtnh-hub/internal/util"
	"github.com/bwmarrin/discordgo"
)

type Discord struct {
	webhookId    string
	webhookToken string
	WebhookChan  string
	avaUrl       string

	sess *discordgo.Session
}

func NewDiscord(token, whId, whToken, avaUrl string) (*Discord, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	wh, err := dg.Webhook(whId)
	if err != nil {
		return nil, err
	}
	slog.Debug("Webhook created", slog.String("channel", wh.ChannelID))

	return &Discord{
		webhookId:    whId,
		webhookToken: whToken,
		WebhookChan:  wh.ChannelID,
		avaUrl:       avaUrl,
		sess:         dg,
	}, nil
}

func (d *Discord) Setup(
	connect func(s *discordgo.Session, m *discordgo.Connect),
	handler func(s *discordgo.Session, m *discordgo.MessageCreate),
	disconnect func(s *discordgo.Session, m *discordgo.Disconnect),
) error {
	d.sess.AddHandler(connect)
	d.sess.AddHandler(handler)
	d.sess.AddHandler(disconnect)
	return d.sess.Open()
}

func (d *Discord) Send(msg chat.Message) error {
	var b strings.Builder

	b.WriteByte('[')
	b.WriteString(msg.Server)
	b.WriteString("] ")
	b.WriteString(util.CleanMinecraftTags(msg.SenderFormatted))
	sender := b.String()

	ava := d.avaUrl + msg.Sender

	p := &discordgo.WebhookParams{
		Content:   msg.Text,
		Username:  sender,
		AvatarURL: ava,
	}

	_, err := d.sess.WebhookExecute(d.webhookId, d.webhookToken, false, p)
	if err != nil {
		return err
	}
	slog.Debug("Discord send",
		slog.String("Username", sender),
		slog.String("Content", msg.Text),
		slog.String("Avatar", ava))
	return nil
}

func (d *Discord) Close() {
	d.sess.Close()
	slog.Info("Discord closed")
}
