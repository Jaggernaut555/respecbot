package logging

import (
	"log"
	"os"
)

var (
	logger *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
}

//Log Log all the given data
func Log(data ...string) {
	logger.Print(data)
}
