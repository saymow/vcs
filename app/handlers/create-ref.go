package handlers

import (
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func CreateRef(name string) {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)

	checkError(repository.CreateRef(name))
}