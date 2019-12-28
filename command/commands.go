package command

import (
	"Jocker/boot"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var RunCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			Jocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		cmd := context.Args().Get(0)
		tty := context.Bool("ti")
		fmt.Println(cmd,tty)
		Run(cmd,tty)
		return nil
	},
}

var InitCommand = cli.Command{
	Name:  "boot",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
		log.Infof("boot come on")
		cmd := context.Args().Get(0)
		log.Infof("command %s", cmd)
		err := boot.RunContainerInitProcess(cmd, nil)
		return err
	},
}