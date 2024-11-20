package handlers

import (
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func Load(name string) {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)

	err = repository.Load(name)

	if _, ok := err.(*repositories.ValidationError); ok {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
