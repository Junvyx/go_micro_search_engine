package util

import (
	"log"
	"os"
)

var Log = log.New(os.Stdout, "[mySEngine]", log.Lshortfile|log.Ldate|log.Ltime)
