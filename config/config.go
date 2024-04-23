package config

import (
	"os"
)

const ConfigPath = "/.config/jirate/config.txt"

func GetConfigFile() (*os.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := home + ConfigPath
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}
