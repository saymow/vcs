package handlers

import (
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

func ShowRefs() {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)
	refs := repository.GetRefs()

	for name, saveName := range refs.Refs {
		if refs.Head == name {
			fmt.Fprint(os.Stdout, "\033[0mHEAD \033[0m-> ")
		}
		fmt.Fprintf(os.Stdout, "\033[34m%s \033[0m-> \033[33m%s\n", name, saveName)
	}

	if _, ok := refs.Refs[refs.Head]; !ok {
		fmt.Fprintf(os.Stdout, "\033[0mHEAD \033[0m-> \033[33m%s\n", refs.Head)
	}
}
