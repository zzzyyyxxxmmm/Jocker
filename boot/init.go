package boot

import (
	"fmt"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func RunContainerInitProcess() error {
	cmdArray:=readUserCommand()
	if cmdArray==nil || len(cmdArray)==0 {
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}
	setUpMount()
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
	cmd.Dir="/root/busybox"
	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

func setUpMount(){
	pwd, err:=os.Getwd()

	if err!=nil{
		log.Errorf("Get current location error %v", err)
	}

	log.Infof("Current location is %s", pwd)
	pivotRoot(pwd)

	defaultMountFlags:=syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags),"")

	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME,"mode=755")
}

func pivotRoot (root string) error {
	if err:=syscall.Mount(root, root, "bind", syscall.MS_BIND | syscall.MS_REC, ""); err!=nil{
		return fmt.Errorf("Mount rootfs to itself error: %v",err)
	}

	pivotDir:=filepath.Join(root, ".pivot_root")

	if err:=os.Mkdir(pivotDir, 0777); err!=nil{
		return err
	}

	if err:=syscall.PivotRoot(root,pivotDir);err!=nil{
		return fmt.Errorf("pivot_root %v", err)
	}

	if err:=syscall.Chdir("/");err!=nil{
		return fmt.Errorf("chdir / %v", err)
	}

	pivotDir= filepath.Join("/",".pivot_root")

	if err:=syscall.Unmount(pivotDir, syscall.MNT_DETACH); err!=nil{
		return fmt.Errorf("unmount pivot_root dir %v",err)
	}
	return os.Remove(pivotDir)
}
