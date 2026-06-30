package game

import (
	"encoding/json"
	"sync"
)

type Instance struct {
	mu        sync.Mutex
	jsonCache []byte

	GameServers map[string]*GameServer `json:"gameServers"`
}

type GameServer struct {
	OnlinePlayers []Player `json:"players"`
	Slots         int      `json:"slots"`
	Mspt          float32  `json:"mspt"`
	Tps           float32  `json:"tps"`
}

func NewInstance() *Instance {
	return &Instance{GameServers: make(map[string]*GameServer)}
}

func (i *Instance) GetJSON() []byte {
	i.mu.Lock()
	defer i.mu.Unlock()
	buf := make([]byte, len(i.jsonCache))
	copy(buf, i.jsonCache)
	return buf
}

func (i *Instance) Copy() (*Instance, error) {
	var new *Instance
	if err := json.Unmarshal(i.GetJSON(), new); err != nil {
		return nil, err
	}
	return new, nil
}

func (i *Instance) GetAllPlayersCount() int {
	count := 0
	for _, v := range i.GameServers {
		count += len(v.OnlinePlayers)
	}
	return count
}

func (i *Instance) SetServer(id string, server *GameServer) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.GameServers[id] = server

	data, err := json.Marshal(i)
	if err != nil {
		return err
	}

	i.jsonCache = data
	return nil
}
