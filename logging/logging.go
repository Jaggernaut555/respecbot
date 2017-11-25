package logging

import (
	"log"
	"os"
)

var (
	logger    *log.Logger
	errLogger *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errLogger = log.New(os.Stderr, "ERR - ", log.Ldate|log.Ltime)
}

//Log Log all the given data
func Log(data ...string) {
	logger.Print(data)
}

//Err Log given data to stderr
func Err(data ...string) {
	errLogger.Print(data)
}
