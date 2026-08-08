package game

import (
	"time"

	"github.com/gorcon/rcon"
)

const RConTimeout = 10 * time.Second

func PingRCon(address, password string) error {
	conn, err := rcon.Dial(address, password,
		rcon.SetDialTimeout(RConTimeout),
		rcon.SetDeadline(RConTimeout))
	if err != nil {
		return err
	}
	return conn.Close()
}
