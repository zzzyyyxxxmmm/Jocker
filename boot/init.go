package boot

import (
	"fmt"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func RunContainerInitProcess() error {
	cmdArray:=readUserCommand()
	if cmdArray==nil || len(cmdArray)==0 {
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	path, err:=exec.LookPath(cmdArray[0])
	if err!=nil{
		log.Errorf("Exec loop path error %v", err)
		return err
	}
	log.Infof("Find path %s",path)
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}

func readUserCommand() []string{
	pipe:=os.NewFile(uintptr(3),"pipe")
	msg,err:=ioutil.ReadAll(pipe)
	if err!=nil{
		log.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr:=string(msg)
	return strings.Split(msgStr," ")
}

func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
	readPipe, writePipe , err :=NewPipe()

	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}

	cmd := exec.Command("/proc/self/exe", "boot")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.ExtraFiles=[]*os.File{readPipe}
	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

