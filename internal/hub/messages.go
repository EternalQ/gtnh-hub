package hub

import (
	"encoding/json"

	"github.com/EternalQ/gtnh-hub/internal/game"
)

const (
	// active acitons
	ActionInit         = "init"
	ActionChat         = "chat"
	ActionPlayerLogged = "player.logged"
)

type Message struct {
	Origin string `json:"origin"`
	Action string `json:"action"`

	Payload json.RawMessage `json:"payload"`
}

type InitMessage struct {
	game.GameServer
}

type ChatMessage struct {
	Sender          string `json:"sender"`
	SenderFormatted string `json:"senderFormatted"`
	Text            string `json:"text"`
}

type PlayerLoggedMessage struct {
	game.Player

	In        bool   `json:"in"`
	Timestamp string `json:"timestamp"`
}
