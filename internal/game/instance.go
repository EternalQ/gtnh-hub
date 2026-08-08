package game

import (
	"maps"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
)

type Instance struct {
	mu sync.RWMutex

	GameServers map[string]*GameServer
}

type GameServer struct {
	IP       string
	RConPort int

	stat atomic.Pointer[ServerStat]
}

type ServerStat struct {
	OnlinePlayers []Player `json:"players"`
	Slots         int      `json:"slots"`
	Mspt          float32  `json:"mspt"`
	Tps           float32  `json:"tps"`
}

func NewInstance() *Instance {
	return &Instance{
		GameServers: make(map[string]*GameServer),
	}
}

func (i *Instance) Connect(id, ip string, rconPort int) *GameServer {
	gs := &GameServer{IP: ip, RConPort: rconPort}

	i.mu.Lock()
	i.GameServers[id] = gs
	i.mu.Unlock()

	return gs
}

func (i *Instance) Disconnect(id string) {
	i.mu.RLock()
	gs, ok := i.GameServers[id]
	i.mu.RUnlock()

	if ok {
		gs.SetStat(nil)
	}
}

func (i *Instance) Snapshot() map[string]*GameServer {
	i.mu.RLock()
	defer i.mu.RUnlock()

	out := make(map[string]*GameServer, len(i.GameServers))
	maps.Copy(out, i.GameServers)
	return out
}

func (i *Instance) ServerIP(id string) (string, bool) {
	i.mu.RLock()
	gs, ok := i.GameServers[id]
	i.mu.RUnlock()

	if !ok {
		return "", false
	}
	return gs.IP, true
}

func (i *Instance) RConAddr(id string) (string, bool) {
	i.mu.RLock()
	gs, ok := i.GameServers[id]
	i.mu.RUnlock()

	if !ok || gs.IP == "" || gs.RConPort == 0 {
		return "", false
	}
	return net.JoinHostPort(gs.IP, strconv.Itoa(gs.RConPort)), true
}

func (i *Instance) AllPlayersCount() int {
	count := 0
	for _, gs := range i.Snapshot() {
		if stat := gs.Stat(); stat != nil {
			count += len(stat.OnlinePlayers)
		}
	}
	return count
}

func (gs *GameServer) Stat() *ServerStat {
	return gs.stat.Load()
}

func (gs *GameServer) SetStat(s *ServerStat) {
	gs.stat.Store(s)
}
