package api

import (
	"os"
	"path/filepath"
)

const MachineUnixSocketName = "machine.socket"

func UserDataDir() (string, error) {
	p, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(p, ".local", "share"), nil
}

func APISocketPath() string {
	udd, err := UserDataDir()
	if err != nil {
		return ""
	}
	return filepath.Join(udd, "machined", MachineUnixSocketName)
}
