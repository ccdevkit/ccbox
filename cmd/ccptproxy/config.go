package main

import (
	"encoding/json"
	"os"
)

// ProxyConfig is the container-side representation of the ccbox-proxy.json
// configuration file written by the host.
type ProxyConfig struct {
	HostAddress string   `json:"hostAddress"`
	Passthrough []string `json:"passthrough"`
	Verbose     bool     `json:"verbose"`
}

// ReadConfig reads and unmarshals a ccbox-proxy.json file at the given path.
func ReadConfig(path string) (*ProxyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ProxyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
