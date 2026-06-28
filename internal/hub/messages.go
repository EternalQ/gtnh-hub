package hub

import (
	"encoding/json"

	"github.com/EternalQ/gtnh-hub/internal/game"
)

// Active acitons
const (
	ActionInfo         = "info"
	ActionChat         = "chat"
	ActionPlayerLogged = "player.logged"
)

type Message struct {
	Origin string `json:"origin"`
	Action string `json:"action"`

	Payload json.RawMessage `json:"payload"`
}

type InfoMessage struct {
	game.GameServer
}

type ChatMessage struct {
	Sender          string `json:"sender"`
	SenderFormatted string `json:"senderFormatted"`
	Text            string `json:"text"`
}

type PlayerLoggedMessage struct {
	game.Player

	In        bool `json:"in"`
	Timestamp int  `json:"timestamp"`
}
