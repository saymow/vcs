package main

import (
	"saymow/version-manager/handlers"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Init struct {
	} `cmd:"" help:"Initializes a repository in the current directory."`
}

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "init":
		handlers.HandleInit()
	default:
		panic(ctx.Command())
	}
}
