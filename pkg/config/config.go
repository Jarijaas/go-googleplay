package config

import (
	"os"
	"path"
)

func GetConfigDirectoryPath() string  {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return path.Join(homeDir, ".gogplay")
}

func GetConfigDirectoryProfilesPath() string  {
	return path.Join(GetConfigDirectoryPath(), "profiles")
}

