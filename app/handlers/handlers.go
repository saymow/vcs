package handlers

import (
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func checkError(err error) {
	if err == nil {
		return
	}

	if _, ok := err.(*repositories.ValidationError); ok {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	errors.Error("unexpected error")
}
