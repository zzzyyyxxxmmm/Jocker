package main

import (
	"Jocker/command"
	"github.com/urfave/cli"
	"log"
	"os"
)
func main() {
	app := cli.NewApp()
	app.Name = "Jocker"
	app.Usage = "Meow~"
	app.Commands = []cli.Command{
		command.RunCommand,
	}

	if err := app.Run(os.Args); err!=nil{
		log.Fatal(err)
	}
}


