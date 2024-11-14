package cmd

import (
	"saymow/version-manager/app/handlers"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Init struct {
	} `cmd:"" help:"Initializes a repository in the current directory."`
	Add struct {
		Paths []string `arg:"" name:"path" help:"list of files paths." type:"path"`
	} `cmd:"" help:"Add files to the index."`
	Rm struct {
		Paths []string `arg:"" name:"path" help:"list of files paths." type:"path"`
	} `cmd:"" help:"Remove files from the index or working directory"`
	Save struct {
		Message string `arg:"" name:"message" help:"Save message."`
	} `cmd:"" help:"Create a save point using the current index."`
	Status struct {
	} `cmd:"" help:"Check the current working status."`
}

func Start() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "init":
		handlers.Init()
	case "add <path>":
		handlers.TrackFiles(ctx.Args[1:])
	case "rm <path>":
		handlers.UntrackFiles(ctx.Args[1:])
	case "save <message>":
		handlers.Save(ctx.Args[1])
	case "status":
		handlers.Status()
	default:
		panic(ctx.Command())
	}
}
