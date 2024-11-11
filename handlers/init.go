package handlers

import (
	"log"
	"os"
	"path"
)

const (
	REPOSITORY_FOLDER_NAME = "repository"
	OBJECT_FOLDER_NAME     = "objects"
	STAGING_AREA_FILE_NAME = "staging"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func HandleInit() {
	currentDir, err := os.Getwd()
	check(err)

	err = os.Mkdir(path.Join(currentDir, REPOSITORY_FOLDER_NAME), 0755)
	check(err)

	stageFile, err := os.Create(path.Join(currentDir, REPOSITORY_FOLDER_NAME, STAGING_AREA_FILE_NAME))
	check(err)

	_, err = stageFile.Write([]byte("Tracked objects:\r\n\r\n"))
	check(err)

	err = stageFile.Close()
	check(err)

	err = os.Mkdir(path.Join(currentDir, REPOSITORY_FOLDER_NAME, OBJECT_FOLDER_NAME), 0755)
	check(err)
}
