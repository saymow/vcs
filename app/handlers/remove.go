package handlers

import (
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func Remove(paths []string) {
	dir, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(dir)

	for _, path := range paths {
		repository.RemoveFile(path)
	}

	repository.SaveIndex()
}
