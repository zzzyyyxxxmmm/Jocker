# 什么是Docker
docker将应用打包成一个镜像, 并且以容器的方式运行. Docker容器将一系列软件包装在一个完整的文件系统中, 这个文件系统包含应用程序运行所需要的一切: 代码, 运行时工具, 系统依赖.

Docker容器具有以下3个特点:
* 轻量级: 在同一台宿主机上的容器共享系统kernel, 这使得它们可以迅速启动而且占用内存极少. 镜像是以分层文件系统构造的, 这可以让他们共享相同的文件, 使得磁盘使用率和镜像下载速度得到提高. 虚拟机会包含一整个操作系统, 容器只包含用户的程序和所有依赖
* 开放: Docker容器基于开放标准, 这使得Docker容器可以运行在主流Linux发行版和Windows操作系统上
* 安全: 容器将各个应用程序隔离开, 这给所有的应用程序提供了一层额外的安全防护

Docker是一个使用了Linux Namespace和Cgroup的虚拟化工具

### 通过cgroup来限制stress进程的资源
1. ```mkdir cgroup-test```
2. ```sudo mount -t cgroup -o none,name=cgroup-test cgroup-test ./cgroup-test```
3. ```ls ./cgroup-test```

```
jikangwang@ubuntu:~/cgroup-test$ ls
cgroup.clone_children  #cpuset的subsystem会读取这个配置文件, 如果这个值是1(默认是0), 子cgroup才会继承父cgroup的cpuset的配置

cgroup.sane_behavior   
cgroup.procs     #cgroup.procs是树中当前节点cgroup的groupid, 现在的位置是在根节点, 这个文件中会有现在系统中所有进程组的ID     
notify_on_release #notify_on_realease标识当这个cgroup最后一个进程退出的时候是否执行了release_agent; release_agent则是一个路径, 通常用作进程退出之后自动清理掉不再使用的cgroup
release_agent     
tasks #标识该cgroup下面的进程ID, 如果把一个进程ID写到tasks文件中, 便会将相应的进程加入到这个cgroup中
```

4. 在上面创建hierarchy的时候, 这个hierarchy并没有关联到任何的subsystem, 所以没办法通过那个hierarchy中的cgroup节点限制进程的资源占用, 其实系统默认已经为每个subsystem创建了一个默认的hierarchy, 比如memory的hierarchy.

```
mount | grep memory
cgroup on /sys/fs/cgroup/memory type cgroup (rw,nosuid,nodev,noexec,relatime,memory)
```
5. ```cd /sys/fs/cgroup/memory```
6. ```stress --vm-bytes 200m --vm-keep -m 1```
7. ```sudo mkdir test-limit-memory && cd test-limit-memory```
8. ```sudo sh -c "echo "100m" > memory.limit_in_bytes"```
9. ```sudo sh -c "echo $$ > tasks"```
10. ```stress --vm-bytes 200m --vm-keep -m 1```

### 用Go语言实现通过cgroup限制容器的资源
我们知道 Docker是通过 Cgroups实现容器资源限制和监控的，下面以一个实际的容器实例来看一下Docker是如何配置 Cgroups 的。
```
docker run -m 设置内存限制

sudo docker run -itd m 128m ubuntu
957459145e9092618837cf94alcb356e206f2f0da560b40cb31035e442d3dfll

docker 会为每个容器在系统的 hierarchy 中创建 cgroup

cd /sys/fs/cgroup/memory/docker/957459145e9092618837cf94alcb356e206f2f0da560b40cb310 35e442d3dfl1

cat memory.limit_in_bytes
134217728

cat memory.usage_in_bytes
430080
```
可以看到, Docker通过为每个容器创建cgroup, 并通过cgroup去配置资源限制和资源监控。

在Namespace容器的基础上, 加上cgroup的限制, 实现一个demo, 使其能够具有限制容器内存的功能
```go
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
)

// 挂载了memory subsystem的hierarchy的根目录位置
const cgroupMemoryHierarchyMount = "/sys/fs/cgroup/memory"

func main() {
	/*
	sudo ls -lt /proc/self/exe
	lrwxrwxrwx 1 root root 0 Dec 26 16:06 /proc/self/exe -> /bin/ls
	正在运行的进程的软连接
	*/
	if os.Args[0] == "/proc/self/exe" {
		// 容器进程
		fmt.Printf("current pid %d\n", syscall.Getpid())
		cmd := exec.Command("sh", "-c", `stress --vm-bytes 200m --vm-keep -m 1`)
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	cmd := exec.Command("/proc/self/exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error", err)
		os.Exit(1)
	} else {
		// 得到fork出来进程映射在外部命名空间的pid
		fmt.Printf("%v\n", cmd.Process.Pid)
		// 在系统默认创建挂载了 memory subsystem 的hierarchy上创建cgroup
		os.Mkdir(path.Join(cgroupMemoryHierarchyMount, "testmemorylimit"), 0755)
		// 将容器进程加入到这个cgroup中
		ioutil.WriteFile(path.Join(cgroupMemoryHierarchyMount, "testmemorylimit", "tasks"), []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
		// 限制cgroup进程使用
		ioutil.WriteFile(path.Join(cgroupMemoryHierarchyMount, "testmemorylimit", "memory.limit_in_bytes"), []byte("100m"), 0644)
		cmd.Process.Wait()
	}
}
```

# UFS
每一个Docker image都是由一系列read-only layer组成的。image layer的内容都存储在Docker hosts filesystem的/var/lib/docker/aufs/diff目录下。而/var/lib/docker/aufs/layers 目录, 则存储着image layer如何堆找这些layer的metadata。

启动一个container的时候, Docker会为其创建一个read-only的init layer, 用来存储与这个容器内环境相关的内容; Docker还会为其创建一个read-write的layer来执行所有的写操作.

container layer的mount目录也是/var/lib/docker/aufs/mnt. container的metadata和配置文件都存放在/var/lib/docker/containers/container-id目录中. container的read-write layer存储在/var/lib/docker/aufs/diff/目录下. 即使容器停止, 这个可读写层依然存在, 因而重启容器不会丢失数据, 只有当一个容器被删除的时候, 这个可读写层才会一起删除.

# 构造容器

## 构造实现run命令版本的容器
```
git clone https://github.com/xianlubird/mydocker.git 
git checkout code-3.1
```

./mydocker run -ti /bin/sh

如果我们直接执行sh命令, 执行ps -ef的时候还是会显示parent process的进程, 因此我们需要在这个进程内重新挂载/proc, 由之前所学我们知道可以在新开的shell里执行```mount -t proc proc /proc```来挂载, 这是一个初始化步骤, 因此我们可以新在进程里新开一个init命令, 挂载完之后再运行shell. 不过, 由于我们不希望init作为第一个进程, 因此可以通过syscall.Exec来覆盖掉当前的init进程.

因此整体流程:
1. ./Jocker run /bin/sh
2. 调用/proc/self/exe 即./Jocker init /bin/sh
3. 挂载, 然后syscall.exec执行sh

## 增加容器资源限制
```mydocker run -ti -m 100m -cpuset 1 -cpushare 512 /bin/sh```

核心:

1. ```sudo mkdir test-limit-memory && cd test-limit-memory```
2. ```sudo sh -c "echo "100m" > memory.limit_in_bytes"```
3. ```sudo sh -c "echo $$ > tasks"```

其实主要步骤就是找到对应的subsystem目录, 然后创建group, 在group里写入限制和pid, 那么如何找到挂载了subsystem的hierarchy的挂载目录的呢? 之前我们是通过```mount | grep memory```来找的, 现在来看看另一种方法:

```
cat /proc/self/mountinfo

41 31 0:36 / /sys/fs/cgroup/rdma rw,nosuid,nodev,noexec,relatime shared:20 - cgroup cgroup rw,rdma
42 31 0:37 / /sys/fs/cgroup/freezer rw,nosuid,nodev,noexec,relatime shared:21 - cgroup cgroup rw,freezer
43 31 0:38 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime shared:22 - cgroup cgroup rw,memory
44 31 0:39 / /sys/fs/cgroup/cpu,cpuacct rw,nosuid,nodev,noexec,relatime shared:23 - cgroup cgroup rw,cpu,cpuacct
45 31 0:40 / /sys/fs/cgroup/cpuset rw,nosuid,nodev,noexec,relatime shared:24 - cgroup cgroup rw,cpuset
46 23 0:41 / /proc/sys/fs/binfmt_misc rw,relatime shared:25 - autofs systemd-1 rw,fd=31,pgrp=1,timeout=0,minproto=5,maxproto=
```

通过这个命令可以找到当前进程相关的mount信息

## 增加管道及环境识别变量
在3.2节中实现的那个简单版本的run命令有一个缺陷，就是传递参数。在父进程和子进程之间传参，是通过调用命令后面跟上参数，也就是/proc/self/exe init args这种方式进行的，然后，在init进程内去解析这个参数，执行相应的命令。但是，这有一个缺点就是，如果用户输入的参数很长，或者其中带有一些特殊字符，那么这种方案就会失败了。其实，runC 实现的方案是通过匿名管道来实现父子进程之间通信的，下面就会修改上一版本的代码，增加这个功能。

```go
func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
```

在创建父进程的时候创建好管道, 运行的时候把读管道通过fd传给子进程

# 构造镜像

## 使用busybox创建容器
```
docker pull busybox
docker run -d busybox top -b
docker export -o busybox.tar 428137198312(容器id)
tar -xvf busybox.tar -C busybox/
```

## 使用AUFS包装busybox
Docker在使用镜像启动一个容器时，会新建2个layer: write layer和container-init layer。 write layer是容器唯一的可读写层: 而container-init layer是为容器新建的只读层，用来存储容器启动时传入的系统信息(前面也提到过，在实际的场景下，它们并不是以write layer container-init layer命名的)。最后把write layer、container-init layer和相关镜像的layers都mount到一个mnt目录下，然后把这个mnt目录作为容器启动的根目录。

在4.1节中己经实现了使用宿主机/root/busybox 目录作为文件的根目录，但在容器内对文件的操作仍然会直接影响到宿主机的/root/busybox目录。本节要进一步进行容器和镜像隔离，实现在容器中进行的操作不会对镜像产生任何影响的功能。 

使用AUFS创建容器文件系统的实现过程如下。启动容器的时候:
1. 创建只读层(busybox) ;
2. 创建容器读写层(writeLayer) ;
3. 创建挂载点(mnt)，井把只读层和读写层挂载到挂载点:
4. 将挂载点作为容器的根目录。 

容器退出的时候:
1. 卸载挂载点(mnt)的文件系统;
1. 删除挂载点; 
2. 删除读写层(writeLayer)。

## 实现volume数据卷
启动一个容器，把宿主机的 /root/volume挂载到容器的 /containerVolume 目录下。

```
./mydocker run -ti -v /root/volume:/containerVolume sh
```

挂载数据卷的过程如下。
1. 首先，读取宿主机文件目录URL，创建宿主机文件目录(/root/${parentUrl})。
2. 然后，读取容器挂载点URL，在容器文件系统里创建挂载点 (/root/rnnt/${containerUrl})。
3. 最后，把宿主机文件目录挂载到容器挂载点。这样启动容器的过程，对数据卷的处理也就完成了。

DeleteMountPointWithVolume函数的处理逻辑如下。
1. 首先，卸载volume挂载点的文件系统(/root/mnt/${containerUrl})，保证整个容器的挂载点没有被使用。
1. 然后，再卸载整个容器文件系统的挂载点 (/root/mnt)。
2. 最后，删除容器文件系统挂载点。整个容器退出过程中的文件系统处理就结束了。

## 实现简单镜像打包
```
./mydocker run -ti sh

./mydocker commit image
```

# 构建容器进阶

## 实现容器的后台运行
容器，在操作系统看来，其实就是一个进程。当前运行命令的 mydocker是主进程，容器是被当前mydocker进程fork出来的子进程。子进程的结束和父进程的运行是一个异步的过程， 即父进程永远不知道子进程到底什么时候结束。如果创建子进程的父进程退出，那么这个子进程就成了没人管的孩子，俗称孤儿进程。为了避免孤儿进程退出时无法释放所占用的资源而僵死，进程号为l的进程init就会接受这些孤儿进程。这在交互式创建 容器的步骤里面是没问题的 ， 但是在这里，如果 etach创建了容器，就不能再去等待，创建容器之后，父进程就已经退出了。因此，这里只是将容器内的init进程启动起来，就己经完成工作，此处添加了判断，原来parent.Wait()主要是用于父进程等待子进程结束，
紧接着就可以退出，然后由操作系统进程ID为1的init进程去接管容器进程。

## 实现查看运行中的容器
将container信息写到config.json

## 实现查看容器日志
将容器进程标准输出挂载到container.log文件中

## 实现进入容器Namespace
setns是一个系统调用，可以根据提供的 PID再次进入到指定的 Namespace 中。它需要先 打开/proc/[pid]/ns/文件夹下对应的文件，然后使当前进程进入到指定的 Namespace 中。系统 调用描述非常简单，但是有一点对于 Go 来说很麻烦。对于 Mount Namespace来说， 一个具有多线程的进程是无法使用setns调用进入到对应的命名空间的。但是， Go每启动一个程序就会进入多线程状态，因此无法简简单单地在 Go里面直接调用系统调用，使当前的进程进入对应的Mount Namespace。这里需要借助C来实现这个功能.

## 实现停止容器
在前面章节创建的容器中， 所有关于容器的信息，比如 PID 、容器创建时间、容器运行命令等，都没有记录，这导致容器运行完后就再也不知道它的信息了，因此需要把这部分信息保留下来。

stopContainer 的主要步骤如下。 
1. 获取容器 PID。
2. 对该 PID发送 kill信号。
3. 修改容器信息。
4. 重新写入存储容器信息的文件。

## 实现删除容器
1. 根据容器名查找容器信息。
2. 判断容器是否处于停止状态。
3. 查找容器存储信息的地址。
4. 移除记录容器信息的文件。

## 实现通过容器制作镜像
1. 为每个容器分配单独的隔离文件
2. 修改mydocker commit命令, 实现对不同容器进行打包镜像的功能

```
sudo ./main run -d --name container1 -v /root/from1:/to1 busybox top
sudo ./main run -d --name container2 -v /root/from2:/to2 busybox top

tree writeLayer

sudo ./main exec container1 sh
echo -e "hello container1" >> /to1/test1.txt

mkdir to1-1
echo -e "hello container1,to1-1,test1" > /to1-1/test1.txt

sudo ./main commit container1 image1

sudo ./main run -d --name container3 -v /root/from1:/to1 container1 top
sudo ./main exec container2 sh
```

# 容器网络
<div align=center>
<img src="https://github.com/zzzyyyxxxmmm/basics/blob/master/image/docker_net_04.png" width="700" height="500">
</div>

<div align=center>
<img src="https://github.com/zzzyyyxxxmmm/basics/blob/master/image/docker_net_1.png" width="500" height="300">
</div>

<div align=center>
<img src="https://github.com/zzzyyyxxxmmm/basics/blob/master/image/docker_net_2.png" width="500" height="300">
</div>

<div align=center>
<img src="https://github.com/zzzyyyxxxmmm/basics/blob/master/image/docker_net_3.png" width="500" height="300">
</div>

## Linux Veth
```
# #创建两个网络 Namespace
ikangwang@ubuntu:~$ sudo ip netns add ns1
jikangwang@ubuntu:~$ sudo ip netns add ns2
# 创建一对 Veth
jikangwang@ubuntu:~$ sudo ip link add veth0 type veth peer name veth1
# 分别将两个Veth移到两个Namespace中
jikangwang@ubuntu:~$ sudo ip link set veth0 netns ns1
jikangwang@ubuntu:~$ sudo ip link set veth1 netns ns2
# 去ns1的namespace中查看网络设备
jikangwang@ubuntu:~$ sudo ip netns exec ns1 ip link
1: lo: <LOOPBACK> mtu 65536 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
5: veth0@if4: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether 0a:5b:5e:d9:e8:79 brd ff:ff:ff:ff:ff:ff link-netnsid 1

# 配置每个 veth 的网络地址和 Namespace 的路由
sudo ip netns exec ns1 ifconfig veth0 172.168.0.2/24 up
sudo ip netns exec ns2 ifconfig veth1 172.168.0.3/24 up
sudo ip netns exec ns1 route add default dev veth0
sudo ip netns exec ns2 route add default dev veth1

sudo ip netns exec ns1 ping -c 1 172.18.0.3
PING 172.18.0.3 (172.18.0.3) 56(84) bytes of data.

--- 172.18.0.3 ping statistics ---
1 packets transmitted, 0 received, 100% packet loss, time 0ms
```

## Linux Bridge
```
# 创建Veth设备并将一端移入Namespace
sudo ip netns add ns1
sudo ip link add veth0 type veth peer name veth1
sudo ip link set veth1 netns ns1 

# 创建网桥
sudo brctl addbr br0
# 挂载网络设备
sudo brctl addif br0 eth0
sudo brctl addif br0 veth0
```

## Linux路由表
路由表是 Linux 内核的 一个模块，通过定义路由表来决定在某个网络 Namespace 中包的流 向，从而定义请求会到哪个网络设备上。
```
# 启动虚拟网络设备, 并设置它在Net Namespace中的Ip地址
sudo ip link set veth0 up
sudo ip link set br0 up
sudo ip netns exec ns1 ifconfig veth1 172.18.0.2/24 up
# 分别设置ns1网络空间的路由和宿主机上的路由
# default代表0.0.0.0/0, 即在Net Namespace中所有流量都经过veth1的网络设备流出
sudo ip netns exec ns1 route add default dev veth1
# 在宿主机上将172.168.0.0/24的网段请求路由到br0的网桥
sudo route add -net 172.18.0.0/24 dev br0
```

## Linux iptables
iptables 中的 MASQUERADE 策略可以将请求包中的源地址转换成一个网络设备的地址， 比如 6.1.2 小节介绍的那个 Namespace 中网络设备的地址是 172.18.0.2，这个地址虽然在宿主机 上可以路由到 brO 的 网桥，但是到达宿主机外部之后，是不知道如何路由到这个IP地址的，
所以如果请求外部地址的话，需要先通过MASQUERADE策略将这个IP转换成宿主机出口网卡的IP
```
# 打开IP转发
sudo sysctl -w net.ipv4.conf.all.forwarding=1
net.ipv4.conf.all.fowarding=1
# 对Namespace中发出的包添加网络地址转换
sudo iptables -t nat -A POSTROUTING -s 172.18.0.0/24 -o eth0 -j MASQUERADE
```

iptables中的DNAT策略也是做网络地址的转换，不过它是要更换目标地址，经常用于将内部网络地址的端口映射到外部去。比如，上面那个例子中的Namespace如果需要提供服务给宿主机之外的应用去请求要怎么办呢?外部应用没办法直接路由到172.18.0.2这个地址，这时候就可以用到 DNAT 策略。
```
# 将到宿主机上80端的请求转发到Namespace的IP上
sudo iptables -t nat -A PREROUTING -p tcp -m tcp --dport 80 -j DNAT --to-destination 172.18.0.2:80
```
这样就可以把宿主机上80端口的TCP请求转发到Namespace中的地址172.18.0.2:80，从而实现外部的应用调用。

### code
```
mydocker network create --subnet 192.168.0.0/24 --driver bridge testbridgenet
mydocker run -ti -p 80:80 --net testbridgenet xxxx
```

