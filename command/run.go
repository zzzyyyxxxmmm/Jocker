package command

import (
	"Jocker/boot"
	log "github.com/sirupsen/logrus"
	"os"
)

func Run(command string, tty bool){
	parent := boot.NewParentProcess(tty, command)
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	parent.Wait()
	os.Exit(-1)
}
