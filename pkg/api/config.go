/*
Copyright © 2022 Ryan Harper <rharper@woxford.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package api

import (
	"fmt"
	"os"
	"path/filepath"
)

type MachineDaemonConfig struct {
	ConfigDirectory string
	DataDirectory   string
}

func DefaultMachineDaemonConfig() *MachineDaemonConfig {
	cfg := MachineDaemonConfig{}
	udd, err := UserDataDir()
	if err != nil {
		panic(fmt.Sprintf("Error getting user data dir: %s", err))
	}
	ucd, err := UserConfigDir()
	if err != nil {
		panic(fmt.Sprintf("Error getting user config dir: %s", err))
	}
	cfg.DataDirectory = filepath.Join(udd, "machine")
	cfg.ConfigDirectory = filepath.Join(ucd, "machine")
	return &cfg
}

func UserDataDir() (string, error) {
	p, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(p, ".local", "share"), nil
}

func UserConfigDir() (string, error) {
	p, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(p, ".config"), nil
}
