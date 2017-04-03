package maptiles

import (
	"os"
)

// ensureDirExists creates directory if it doesnt exist
func ensureDirExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0755)
	}
}
