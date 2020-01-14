package command

import (
	"Jocker/boot"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func Run( tty bool, comArray []string , volume string){
	parent, writePipe := boot.NewParentProcess(tty, volume)
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	sendInitCommand(comArray, writePipe)
	parent.Wait()
	mntURL:="/root/mnt/"
	rootURL:="/root/"
	boot.DeleteWorkSpace(rootURL,mntURL,volume)
	os.Exit(-1)
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

