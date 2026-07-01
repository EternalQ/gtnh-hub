package discord

import (
	"log/slog"
	"strings"

	"github.com/EternalQ/gtnh-hub/internal/hub"
	"github.com/EternalQ/gtnh-hub/internal/util"
	"github.com/bwmarrin/discordgo"
)

type Discord struct {
	WebhookChan string
	AdminRoleId string
	PinnedMsg   string

	webhookId    string
	webhookToken string
	avaUrl       string

	sess *discordgo.Session
}

func NewDiscord(token, whId, whToken, avaUrl, adminRoleId, pinnedMsg string) (*Discord, error) {
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
		AdminRoleId:  adminRoleId,
		WebhookChan:  wh.ChannelID,
		webhookId:    whId,
		webhookToken: whToken,
		avaUrl:       avaUrl,
		sess:         dg,
	}, nil
}

func (d *Discord) Setup(
	connect func(s *discordgo.Session, m *discordgo.Connect),
	handler func(s *discordgo.Session, m *discordgo.MessageCreate),
	disconnect func(s *discordgo.Session, m *discordgo.Disconnect),
	refresher func(s *discordgo.Session, chanId, msggId string),
) error {
	d.sess.AddHandler(connect)
	d.sess.AddHandler(handler)
	d.sess.AddHandler(disconnect)
	go refresher(d.sess, d.WebhookChan, d.PinnedMsg)
	return d.sess.Open()
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

	ava := d.avaUrl + msg.Sender

	p := &discordgo.WebhookParams{
		Content:   util.CleanMinecraftTags(msg.Text),
		Username:  sender,
		AvatarURL: ava,
	}

	_, err := d.sess.WebhookExecute(d.webhookId, d.webhookToken, false, p)
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
