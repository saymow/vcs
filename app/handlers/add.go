package handlers

import (
	"os"
)

func HandleTrackFiles(paths []string) {
	dir, err := os.Getwd()
	check(err)

	repository := getRepository(dir)

	for _, path := range paths {
		repository.StageFile(path)
	}

	repository.SaveStagedFiles()
}
