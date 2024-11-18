package handlers

import (
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories"
)

// Wed, Nov 18, 2024, 2:35 PM
const DATE_LAYOUT = "Mon, Jan 06, 2006, 3:04 PM"

func Logs() {
	root, err := os.Getwd()
	errors.Check(err)

	repository := repositories.GetRepository(root)
	logs := repository.GetLogs()

	for _, log := range logs {
		fmt.Fprintf(os.Stdout, "\033[33m %s \033[0m %s \033[32m %s\n", log.Checkpoint.Id, log.Checkpoint.Message, log.Checkpoint.CreatedAt.Format(DATE_LAYOUT))
	}
}
