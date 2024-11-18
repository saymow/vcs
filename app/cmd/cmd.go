package cmd

import (
	"saymow/version-manager/app/handlers"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Init struct {
	} `cmd:"" help:"Initialize a repository in the current directory."`
	Add struct {
		Paths []string `arg:"" name:"path" help:"List of files paths." type:"path"`
	} `cmd:"" help:"Add files to the index."`
	Rm struct {
		Paths []string `arg:"" name:"path" help:"List of files paths." type:"path"`
	} `cmd:"" help:"Remove files from the index and working directory."`
	Save struct {
		Message string `arg:"" name:"message" help:"Save message."`
	} `cmd:"" help:"Create a save point."`
	Status struct {
	} `cmd:"" help:"Check the repository status."`
	Restore struct {
		Path string `arg:"" name:"path" help:"Path to be restored."`
	} `cmd:"" help:"Restore files from index or file tree."`
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
	case "restore <path>":
		handlers.Restore(ctx.Args[1])
	case "status":
		handlers.Status()
	default:
		panic(ctx.Command())
	}
}
