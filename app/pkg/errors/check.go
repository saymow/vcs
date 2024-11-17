package errors

import (
	"log"
	"runtime/debug"
)

func Check(err error) {
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
}

func Error(message string) {
	debug.PrintStack()
	log.Fatal(message)
}
