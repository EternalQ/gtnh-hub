package discord

import (
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
	p := &discordgo.WebhookParams{
		Content:   msg.Text,
		Username:  msg.Sender,
		AvatarURL: d.avaUrl + msg.Sender,
	}

	_, err := d.sess.WebhookExecute(d.webhookId, d.webhookToken, true, p, nil)
	if err != nil {
		return err
	}
	return nil
}

func (d *Discord) Close() {
	d.sess.Close()
}
