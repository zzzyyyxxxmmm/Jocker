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
		cli.StringFlag{
				Name: "v",
				Usage: "volume",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		var cmdArray []string
		for _, arg:=range context.Args(){
			cmdArray=append(cmdArray,arg)
		}
		tty := context.Bool("ti")
		volume := context.String("v")
		detach:=context.Bool("d")

		if tty && detach{
			return fmt.Errorf("ti and d parameter can not both provided")
		}
		Run(tty, cmdArray, volume)
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
		err := boot.RunContainerInitProcess()
		return err
	},
}

var CommitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		imageName := context.Args().Get(0)
		//commitContainer(containerName)
		CommitContainer(imageName)
		return nil
	},
}
