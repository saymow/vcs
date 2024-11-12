package handlers

import (
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func Init() {
	currentDir, err := os.Getwd()
	errors.Check(err)

	repositories.CreateRepository(currentDir)
}
