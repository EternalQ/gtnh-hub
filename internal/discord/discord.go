package discord

import (
	"log/slog"
	"strings"

	"github.com/EternalQ/gtnh-hub/internal/chat"
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
	if err := dg.Open(); err != nil {
		return nil, err
	}

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
	ready func(s *discordgo.Session, m *discordgo.Ready),
	handler func(s *discordgo.Session, m *discordgo.MessageCreate),
	disconnect func(s *discordgo.Session, m *discordgo.Disconnect),
) {
	d.sess.AddHandler(ready)
	d.sess.AddHandler(handler)
	d.sess.AddHandler(disconnect)
}

func (d *Discord) Send(msg chat.Message) error {
	var b strings.Builder

	b.WriteByte('[')
	b.WriteString(msg.Server)
	b.WriteString("] ")
	b.WriteString(msg.SenderFormatted)

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
