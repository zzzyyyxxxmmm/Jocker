package command

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func Run(command string, tty bool){
	cmd:=exec.Command(command)

	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
		syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,}

	if tty {
		cmd.Stdin=os.Stdin
		cmd.Stdout=os.Stdout
		cmd.Stderr=os.Stderr
	}

	if err:=cmd.Start();err!=nil{
		log.Fatal(err)
	}
}
