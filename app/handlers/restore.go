package handlers

import (
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func Restore(path string, ref string) {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)
	err = repository.Restore(ref, path)

	if err != nil {
		if _, ok := err.(*repositories.ValidationError); ok {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		errors.Error(err.Error())
	}
}
