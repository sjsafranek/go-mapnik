package maptiles

import (
	"log"
	"os"
)

var logger *log.Logger = log.New(os.Stdout, "[TILE] ", log.LUTC|log.Ldate|log.Ltime|log.Lshortfile|log.Lmicroseconds)
