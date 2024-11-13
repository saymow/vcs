package errors

import "log"

func Check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Error(message string) {
	log.Fatal(message)
}
