package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config struct for the configuration file prusa.yml
type Config struct {
	Printers []Printers `yaml:"printers"`
}

// Printers struct containing the printer configuration
type Printers struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// LoadConfig function to load and parse the configuration file
func LoadConfig(path string) (Config, error) {
	var config Config
	file, err := os.ReadFile(path)

	if err != nil {
		return config, err
	}

	if err := yaml.Unmarshal(file, &config); err != nil {
		return config, err
	}

	return config, err
}

func getPassword(address string, printers []Printers) string {
	for _, printer := range printers {
		if printer.Address == address {
			return printer.Password
		}
	}
	return ""
}
func getUsername(address string, printers []Printers) string {
	for _, printer := range printers {
		if printer.Address == address {
			return printer.Username
		}
	}
	return ""
}
