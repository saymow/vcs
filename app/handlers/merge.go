package handlers

import (
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func Merge(name string) {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)
	_, err = repository.Merge(name)
	errors.Check(err)

	// Reload the file tree
	repository = repositories.GetRepository(root)
	status := repository.GetStatus()

	fmt.Printf("Ref \"%s\" merged succesfully.\n", name)

	if status.HasChanges() {
		fmt.Println("\nBut you have conflicts to resolve.")
		printStatus(status)
	}

	checkError(err)
}
