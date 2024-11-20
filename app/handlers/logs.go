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
	log := repository.GetLogs()

	for _, saveLog := range log.History {
		fmt.Fprintf(os.Stdout, "\033[33m %s ", saveLog.Checkpoint.Id)

		if len(saveLog.Refs) > 0 {
			fmt.Fprint(os.Stdout, "\033[34m(")

			for idx, ref := range saveLog.Refs {
				if ref == log.Head {
					fmt.Fprintf(os.Stdout, "HEAD -> %s", ref)
				} else {
					fmt.Fprintf(os.Stdout, "%s", ref)
				}

				if idx < len(saveLog.Refs)-1 {
					fmt.Fprint(os.Stdout, ", ")
				}
			}

			fmt.Fprint(os.Stdout, ")")
		}

		fmt.Fprintf(os.Stdout, "\033[0m %s ", saveLog.Checkpoint.Message)
		fmt.Fprintf(os.Stdout, "\033[32m %s\n", saveLog.Checkpoint.CreatedAt.Format(DATE_LAYOUT))
	}
}
