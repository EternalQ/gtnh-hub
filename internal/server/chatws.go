package server

import (
	"fmt"
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
		fmt.Printf("Error while upgrating connection: %s\n", err.Error())
		return
	}

	// get greeting
	msg := &chat.Message{}
	if err := conn.ReadJSON(msg); err != nil {
		fmt.Printf("Error while reading greeting message: %s\n", err.Error())
		conn.Close()
	}
	id := msg.Server

	ch := make(chan chat.Message, 100)

	go func() {
		for msg := range ch {
			if err := conn.WriteJSON(msg); err != nil {
				fmt.Printf("Error while writing message to server (%s): %s\n", id, err.Error())
			}
		}
		conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shutdown"),
		)
		fmt.Printf("Channel closed: %s\n", id)
	}()

	go func() {
		defer s.chat.Unregister(id)
		defer conn.Close()
		for {
			var msg chat.Message
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsCloseError(
					err,
					websocket.CloseGoingAway,
					websocket.CloseNormalClosure,
					websocket.CloseNoStatusReceived,
				) {
					fmt.Printf("Connection %s closed: %s\n", id, err.Error())
				} else {
					fmt.Printf("Error while reading server (%s) message: %s\n", id, err.Error())
				}
				return
			}
			s.chat.Send(msg)
		}
	}()

	s.chat.Register(msg.Server, ch)
}
