package handlers

import (
	"os"
)

func Init() {
	currentDir, err := os.Getwd()
	check(err)

	createRepository(currentDir)
}
