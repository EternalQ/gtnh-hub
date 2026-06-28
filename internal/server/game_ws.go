package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/EternalQ/gtnh-hub/internal/hub"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleGtnh(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Upgrating connection", slog.String("err", err.Error()))
		return
	}

	msg := &hub.Message{}
	if err := conn.ReadJSON(msg); err != nil {
		slog.Error("GTNH-ws reading first message", slog.String("err", err.Error()))
		conn.Close()
	}

	if msg.Action != hub.ActionInfo {
		slog.Error("GTNH-ws first not-init message",
			slog.String("origin", msg.Origin),
			slog.String("action", msg.Action),
		)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "first message should be info"))
		return
	}

	var p hub.InfoMessage
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		slog.Error("GTNH-ws init unmarshal", slog.String("err", err.Error()))
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4002, "bad payload"))
		return
	}
	id := msg.Origin
	s.game.GameServers[id] = &p.GameServer

	// TODO: take message from config
	s.hub.SendRaw(id, hub.ActionChat, &hub.ChatMessage{Sender: "", Text: "[" + id + "] сервер включился!"})

	ch := make(chan hub.Message, 100)

	go func() {
		for msg := range ch {
			if err := conn.WriteJSON(msg); err != nil {
				slog.Error("GTNH-ws writing message",
					slog.String("origin", msg.Origin),
					slog.String("action", msg.Action),
					slog.String("err", err.Error()),
				)
			}
		}
		conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shutdown"),
		)
	}()

	go func() {
		defer func() {
			s.game.GameServers[id] = nil
			s.hub.Unregister(id)
			conn.Close()
		}()

		for {
			var msg hub.Message
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsCloseError(
					err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived,
				) {
					slog.Info("Connection closed", slog.String("origin", id), slog.String("reason", err.Error()))
					s.hub.SendRaw(
						id,
						hub.ActionChat,
						&hub.ChatMessage{Sender: "", Text: "[" + id + "] сервер выключился"},
					)
				} else {
					slog.Error("GTNH-ws reading message",
						slog.String("origin", id),
						slog.String("err", err.Error()),
					)
					s.hub.SendRaw(
						id,
						hub.ActionChat,
						&hub.ChatMessage{Sender: "", Text: "[" + id + "] сервер (или соединение) крашнулся("},
					)
				}
				return
			}

			switch msg.Action {
			case hub.ActionChat:
				s.hub.Send(msg)
			case hub.ActionPlayerLogged:
				s.hub.Send(msg)

				var p hub.PlayerLoggedMessage
				if err := json.Unmarshal(msg.Payload, &p); err != nil {
					slog.Error("GTNH-ws payload unmarshal",
						slog.String("err", err.Error()),
						slog.String("action", msg.Action),
					)
					conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4002, "bad payload"))
					return
				}

				if p.In {
					s.game.AddPlayer(id, &p.Player)
				} else {
					s.game.RemovePlayer(id, &p.Player)
				}
			default:
				slog.Warn("GTNH-ws unimplemented action handler", slog.String("action", msg.Action))
			}
		}
	}()

	s.hub.Register(id, ch)
}
