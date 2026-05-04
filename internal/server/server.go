package server

import (
	"fmt"
	"strings"

	"github.com/EternalQ/gtnh-hub/internal/chat"
	"github.com/EternalQ/gtnh-hub/internal/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
)

type Server struct {
	r    *mux.Router
	chat *chat.Hub
	ds   *discord.Discord
}

func NewServer(ds *discord.Discord, hub *chat.Hub) *Server {
	s := &Server{
		chat: hub,
		r:    mux.NewRouter(),
		ds:   ds,
	}

	ds.Setup(s.dsReady, s.dsHandler, s.dsDisconnect)

	return s
}

func (s *Server) Routes() *mux.Router {
	s.r.HandleFunc("/gtnh-chat", s.handleChat)

	return s.r
}

func (s *Server) dsReady(sess *discordgo.Session, m *discordgo.Ready) {
	ch := make(chan chat.Message, 100)

	go func() {
		for msg := range ch {
			if err := s.ds.Send(msg); err != nil {
				fmt.Printf("Discord message send error: %s\n", err.Error())
			}
		}
		fmt.Printf("Channel closed: Discord\n")
	}()

	s.chat.Register("Discord", ch)
}

func (s *Server) dsHandler(sess *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == sess.State.User.ID || m.ChannelID != s.ds.WebhookChan {
		return
	}
	if strings.HasPrefix(m.Content, "!ping") {
		sess.ChannelMessageSend(m.ChannelID, "Pong! 🏓")
		return
	}

	s.chat.Send(chat.Message{
		Server: "Discord",
		Sender: m.Author.GlobalName,
		Text:   m.Content,
	})
}

func (s *Server) dsDisconnect(sess *discordgo.Session, m *discordgo.Disconnect) {
	s.chat.Unregister("Discord")
}
