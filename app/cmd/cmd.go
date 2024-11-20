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
	} `cmd:"" help:"Show the repository status."`
	Restore struct {
		Path string `arg:"" name:"path" help:"Path to be restored."`
	} `cmd:"" help:"Restore files from index or file tree."`
	Logs struct {
	} `cmd:"" help:"Show the repository saves logs."`
	Refs struct {
	} `cmd:"" help:"Show the repository saves refs."`
	Ref struct {
		Name string `arg:"" name:"name" help:"Reference name."`
	} `cmd:"" help:"Create a reference in the current Save point."`
}

func Start() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "init":
		handlers.Init()
	case "status":
		handlers.ShowStatus()
	case "logs":
		handlers.ShowLogs()
	case "refs":
		handlers.ShowRefs()
	case "add <path>":
		handlers.TrackFiles(ctx.Args[1:])
	case "rm <path>":
		handlers.UntrackFiles(ctx.Args[1:])
	case "save <message>":
		handlers.Save(ctx.Args[1])
	case "restore <path>":
		handlers.Restore(ctx.Args[1])
	case "ref <name>":
		handlers.CreateRef(ctx.Args[1])
	default:
		panic(ctx.Command())
	}
}
