package handlers

import "os"

func UntrackFiles(paths []string) {
	dir, err := os.Getwd()
	check(err)

	repository := getRepository(dir)

	for _, path := range paths {
		repository.RemoveFileIndex(path)
	}

	repository.SaveIndex()
}
