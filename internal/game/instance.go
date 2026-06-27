package game

type Instance struct {
	GameServers map[string]*GameServer
}

type GameServer struct {
	OnlinePlayers []*Player `json:"players"`
	Slots         int       `json:"slots"`
}

func NewInstance() *Instance {
	return &Instance{GameServers: make(map[string]*GameServer)}
}

func (i *Instance) GetAllPlayersCount() int {
	count := 0
	for _, v := range i.GameServers {
		count += len(v.OnlinePlayers)
	}
	return count
}

func (i *Instance) AddPlayer(id string, player *Player) {
	if i.GameServers[id].OnlinePlayers == nil {
		i.GameServers[id].OnlinePlayers = make([]*Player, 8)
	}
	i.GameServers[id].OnlinePlayers = append(i.GameServers[id].OnlinePlayers, player)
}

func (in *Instance) RemovePlayer(id string, player *Player) {
	for i, p := range in.GameServers[id].OnlinePlayers {
		if p.Name == player.Name {
			in.GameServers[id].OnlinePlayers = append(in.GameServers[id].OnlinePlayers[:i], in.GameServers[id].OnlinePlayers[i+1:]...)
		}
	}
}
