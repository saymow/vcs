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
		Ref  string `optional:"" short="r" default:"HEAD" name:"ref" help:The Ref or Save hash to restore from. If omitted, HEAD is used.`
		Path string `arg:"" name:"path" help:"Path to be restored."`
	} `cmd:"" help:"Restore files from index or file tree."`
	Logs struct {
	} `cmd:"" help:"Show the repository saves logs."`
	Refs struct {
	} `cmd:"" help:"Show the repository saves refs."`
	Ref struct {
		Name string `arg:"" name:"name" help:"Reference name."`
	} `cmd:"" help:"Create a reference in the current Save point."`
	Load struct {
		Name string `arg:"" name:"name" help:"Reference name or Save hash."`
	} `cmd:"" help:"Load the file tree of a Ref name or Save hash."`
	Merge struct {
		Name string `arg:"" name:"name" help:"Reference name."`
	} `cmd:"" help:"Merge the file tree of a Ref name."`
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
		handlers.Add(ctx.Args[1:])
	case "rm <path>":
		handlers.Remove(ctx.Args[1:])
	case "save <message>":
		handlers.Save(ctx.Args[1])
	case "restore <path>":
		handlers.Restore(CLI.Restore.Path, CLI.Restore.Ref)
	case "ref <name>":
		handlers.CreateRef(ctx.Args[1])
	case "load <name>":
		handlers.Load(ctx.Args[1])
	case "merge <name>":
		handlers.Merge(ctx.Args[1])
	default:
		panic(ctx.Command())
	}
}
