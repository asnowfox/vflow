package vlogger
import (
	"log"
	"os"
)

var Logger = log.New(os.Stderr, "[vflow] ", log.Ldate|log.Ltime)
