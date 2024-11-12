package main

import (
	"saymow/version-manager/handlers"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Init struct {
	} `cmd:"" help:"Initializes a repository in the current directory."`
	Add struct {
		Paths []string `arg:"" name:"path" help:"list of files paths." type:"path"`
	} `cmd:"" help:"Add files to the stating area."`
}

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "init":
		handlers.HandleInit()
	case "add <path>":
		handlers.HandleTrackFiles(ctx.Args[1:])
	default:
		panic(ctx.Command())
	}
}
