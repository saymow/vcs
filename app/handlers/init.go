package handlers

import (
	"os"
)

func HandleInit() {
	currentDir, err := os.Getwd()
	check(err)

	createRepository(currentDir)
}
