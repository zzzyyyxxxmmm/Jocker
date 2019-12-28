package main

import (
	"Jocker/command"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)
func main() {
	app := cli.NewApp()
	app.Name = "Jocker"
	app.Usage = "Meow~"
	app.Commands = []cli.Command{
		command.RunCommand,
		command.InitCommand,
	}

	app.Before = func(context *cli.Context) error {
		// Log as JSON instead of the default ASCII formatter.
		log.SetFormatter(&log.JSONFormatter{})

		log.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err!=nil{
		log.Fatal(err)
	}
}


