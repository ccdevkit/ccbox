package cmdpassthrough

import (
	"encoding/json"

	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/session"
)

// ProxyConfig is the configuration written for ccptproxy inside the container.
type ProxyConfig struct {
	HostAddress string   `json:"hostAddress"`
	Passthrough []string `json:"passthrough"`
	Verbose     bool     `json:"verbose"`
}

// WriteProxyConfig marshals config to JSON and writes it into the session
// at the container path expected by ccptproxy.
func WriteProxyConfig(sess *session.Session, config ProxyConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return sess.FileWriter.WriteFile(constants.ProxyConfigContainerPath, data, true)
}
