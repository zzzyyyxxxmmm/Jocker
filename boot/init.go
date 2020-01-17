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
var (
	RUNNING             string = "running"
	STOP                string = "stopped"
	Exit                string = "exited"
	DefaultInfoLocation string = "/var/run/mydocker/%s/"
	ConfigName          string = "config.json"
)

type ContainerInfo struct {
	Pid         string `json:"pid"` //容器的init进程在宿主机上的 PID
	Id          string `json:"id"`  //容器Id
	Name        string `json:"name"`  //容器名
	Command     string `json:"command"`    //容器内init运行命令
	CreatedTime string `json:"createTime"` //创建时间
	Status      string `json:"status"`     //容器的状态
}

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

func NewParentProcess(tty bool, volume string) (*exec.Cmd, *os.File) {
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
	mntURL:="/root/mnt/"
	rootURL:="/root/"
	NewWorkSpace(rootURL, mntURL,volume)
	cmd.Dir=mntURL
	return cmd, writePipe
}



func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
//Create a AUFS filesystem as container root workspace
func NewWorkSpace(rootURL string, mntURL string, volume string) {
	CreateReadOnlyLayer(rootURL)
	CreateWriteLayer(rootURL)
	CreateMountPoint(rootURL, mntURL)
	if(volume != ""){
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if(length == 2 && volumeURLs[0] != "" && volumeURLs[1] !=""){
			MountVolume(rootURL, mntURL, volumeURLs)
			log.Infof("%q",volumeURLs)
		}else{
			log.Infof("Volume parameter input is not correct.")
		}
	}
}
func MountVolume(rootURL string, mntURL string, volumeURLs []string)  {
	parentUrl := volumeURLs[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {
		log.Infof("Mkdir parent dir %s error. %v", parentUrl, err)
	}
	containerUrl := volumeURLs[1]
	containerVolumeURL := mntURL + containerUrl
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {
		log.Infof("Mkdir container dir %s error. %v", containerVolumeURL, err)
	}
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Mount volume failed. %v", err)
	}

}


func volumeUrlExtract(volume string)([]string){
	var volumeURLs []string
	volumeURLs =  strings.Split(volume, ":")
	return volumeURLs
}

func CreateReadOnlyLayer(rootURL string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	exist, err := PathExists(busyboxURL)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", busyboxURL, err)
	}
	if exist == false {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("Untar dir %s error %v", busyboxURL, err)
		}
	}
}

func CreateWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.Mkdir(writeURL, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", writeURL, err)
	}
}

func CreateMountPoint(rootURL string, mntURL string) {
	if err := os.Mkdir(mntURL, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", mntURL, err)
	}
	dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	log.Info(cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
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

	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("make parent mount private error: %v", err)
	}

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
//Delete the AUFS filesystem while container exit
func DeleteWorkSpace(rootURL string, mntURL string, volume string) {
	if(volume != ""){
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if(length == 2 && volumeURLs[0] != "" && volumeURLs[1] !=""){
			DeleteMountPointWithVolume(rootURL, mntURL, volumeURLs)
		}else{
			DeleteMountPoint(rootURL, mntURL)
		}
	}else {
		DeleteMountPoint(rootURL, mntURL)
	}
	DeleteWriteLayer(rootURL)
}

func DeleteMountPointWithVolume(rootURL string, mntURL string, volumeURLs []string){
	containerUrl := mntURL + volumeURLs[1]
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout=os.Stdout
	cmd.Stderr=os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount volume failed. %v",err)
	}

	cmd = exec.Command("umount", mntURL)
	cmd.Stdout=os.Stdout
	cmd.Stderr=os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount mountpoint failed. %v",err)
	}

	if err := os.RemoveAll(mntURL); err != nil {
		log.Infof("Remove mountpoint dir %s error %v", mntURL, err)
	}
}

func DeleteMountPoint(rootURL string, mntURL string){
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout=os.Stdout
	cmd.Stderr=os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v",err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("Remove dir %s error %v", mntURL, err)
	}
}

func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.RemoveAll(writeURL); err != nil {
		log.Errorf("Remove dir %s error %v", writeURL, err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

