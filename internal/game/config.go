package game

import "strings"

type GameConfig struct {
	MCSBaseURL   string            `yaml:"mcs_base_url" env:"MCS_BASE_URL" env-default:"http://localhost:23333"`
	MCSAPIKey    string            `yaml:"mcs_api_key" env:"MCS_API_KEY"`
	RConPassword string            `yaml:"rcon_password" env:"RCon_PASSWORD"`
	Instances    map[string]string `yaml:"instances"`
}

func (c GameConfig) ResolveInstance(tag string) (instanceID, daemonID string, ok bool) {
	raw, exists := c.Instances[tag]
	if !exists {
		return "", "", false
	}

	instanceID, daemonID, ok = strings.Cut(raw, ",")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(instanceID), strings.TrimSpace(daemonID), true
}
