package server

import (
	"log/slog"
	"net/http"

	"github.com/EternalQ/gtnh-hub/internal/chat"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Upgrating connection", slog.String("err", err.Error()))
		return
	}

	// get greeting
	msg := &chat.Message{}
	if err := conn.ReadJSON(msg); err != nil {
		slog.Error("Reading ws greeting message", slog.String("err", err.Error()))
		conn.Close()
	}
	id := msg.Server
	// TODO: take message from config
	s.chat.Send(chat.Message{Server: id, Sender: "", Text: "[" + id + "] сервер включился!"})

	ch := make(chan chat.Message, 100)

	go func() {
		for msg := range ch {
			if err := conn.WriteJSON(msg); err != nil {
				slog.Error("Writing ws message",
					slog.String("server", msg.Server),
					slog.String("err", err.Error()))
			}
		}
		conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shutdown"),
		)
	}()

	go func() {
		defer s.chat.Unregister(id)
		defer conn.Close()
		for {
			var msg chat.Message
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsCloseError(
					err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived,
				) {
					slog.Info("Connection closed", slog.String("id", id), slog.String("err", err.Error()))
					s.chat.Send(chat.Message{Server: id, Sender: "", Text: "[" + id + "] сервер выключился"})
				} else {
					slog.Error("Reading ws message", slog.String("server", id), slog.String("err", err.Error()))
					s.chat.Send(chat.Message{Server: id, Sender: "", Text: "[" + id + "] сервер (или соединение) крашнулся("})
				}
				return
			}
			s.chat.Send(msg)
		}
	}()

	s.chat.Register(msg.Server, ch)
}
