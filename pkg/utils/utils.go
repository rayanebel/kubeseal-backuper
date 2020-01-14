package utils

import (
	"os"
)

// HomeDir - Utils to find $HOME path on windows and linux
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
