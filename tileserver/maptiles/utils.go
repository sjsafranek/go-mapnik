package maptiles

import (
	"math/rand"
	"os"
	"time"
)

// ensureDirExists creates directory if it doesnt exist.
func ensureDirExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0755)
	}
}

// RandomIntBetween generates random int between min and max.
func RandomIntBetween(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
