package main

import (
	"log"
	"os"
	"path"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Init struct {
	} `cmd:"" help:"Initializes a repository in the current directory."`
}

const REPOSITORY_FOLDER_NAME = "repository"
const OBJECT_FOLDER_NAME = "objects"
const STAGING_AREA_FILE_NAME = "staging"

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func handleInit() {
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

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "init":
		handleInit()
	default:
		panic(ctx.Command())
	}
}
