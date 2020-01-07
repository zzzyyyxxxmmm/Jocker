package command

import (
	"Jocker/boot"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func Run( tty bool, comArray []string){
	parent, writePipe := boot.NewParentProcess(tty)
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	sendInitCommand(comArray, writePipe)
	parent.Wait()
	os.Exit(-1)
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

