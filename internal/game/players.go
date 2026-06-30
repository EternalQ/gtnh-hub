package game

type Player struct {
	Name          string `json:"name"`
	NameFormatted string `json:"nameFormatted"`
	Uuid          string `json:"uuid"`
	Team          string `json:"team"`
}
