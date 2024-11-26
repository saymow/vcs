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
		Message string `short="m" name:"message" help:"Save message."`
	} `cmd:"" help:"Create a save point with the current index."`
	Status struct {
	} `cmd:"" help:"Show the index and working directory status."`
	Restore struct {
		Ref  string `optional:"" short="r" default:"HEAD" name:"ref" help:"The Ref or Save hash to restore from. If omitted, HEAD is used."`
		Path string `arg:"" name:"path" help:"Path to be restored."`
	} `cmd:"" help:"Restore files from index or file tree.\n\nRestore cover 2 usecases: \n\n 1. Restore HEAD + index (...and remove the index change). \n\n It can be used to restore the current head + index changes. Index changes have higher priorities. \n Initialy Restore will look for your change in the index, if found, the index change is applied. Otherwise, \n Restore will apply the HEAD changes. \n\n 2. Restore Save \n\n It can be used to restore existing Saves to the current working directory. \n\nCaveats: \n\n - Restore will remove the existing changes in the path (forever) and restore reference. \n\n - You can use Restore to recover a deleted file from the index or from a Save. \n\n - The HEAD is not changed during Restore."`
	Logs struct {
	} `cmd:"" help:"Show the repository saves logs."`
	Refs struct {
	} `cmd:"" help:"Show the repository saves refs."`
	Ref struct {
		Name string `short:"n" name:"name" help:"Reference name."`
	} `cmd:"" help:"Create a reference in the current Save point."`
	Load struct {
		Name string `arg:"" name:"name" help:"Reference name or Save hash."`
	} `cmd:"" help:"Load the files tree to the current working directory. HEAD is updated accordingly with name."`
	Merge struct {
		Name string `arg:"" name:"name" help:"Reference name."`
	} `cmd:"" help:"Merge name files tree to the current file tree."`
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
		handlers.Add(CLI.Add.Paths)
	case "rm <path>":
		handlers.Remove(CLI.Rm.Paths)
	case "save":
		handlers.Save(CLI.Save.Message)
	case "restore <path>":
		handlers.Restore(CLI.Restore.Path, CLI.Restore.Ref)
	case "ref":
		handlers.CreateRef(CLI.Ref.Name)
	case "load <name>":
		handlers.Load(CLI.Load.Name)
	case "merge <name>":
		handlers.Merge(CLI.Merge.Name)
	default:
		panic(ctx.Command())
	}
}
