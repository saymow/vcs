package handlers

import (
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func CreateRef(name string) {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)

	err = repository.CreateRef(name)

	if _, ok := err.(*repositories.ValidationError); ok {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func ShowRefs() {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)

	for name, saveName := range repository.GetRefs() {
		fmt.Fprintf(os.Stdout, "\033[34m%s \033[0m-> \033[33m%s\n", name, saveName)
	}
}
