package main

import (
	"github.com/blbecker/webmentionR/cmd/fetch"
	"github.com/urfave/cli/v2"
	"os"

	"github.com/charmbracelet/log"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			&fetch.Command,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
