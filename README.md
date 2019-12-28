# Jocker

A simple container.

# Log

## 构造实现run命令版本的容器
如果我们直接执行sh命令, 执行ps -ef的时候还是会显示parent process的进程, 因此我们需要在这个进程内重新挂载/proc, 由之前所学我们知道可以在新开的shell里执行```mount -t proc proc /proc```来挂载, 这是一个初始化步骤, 因此我们可以新在进程里新开一个init命令, 挂载完之后再运行shell. 
不过, 由于我们不希望init作为第一个进程, 
因此可以通过syscall.Exec来覆盖掉当前的init进程.

因此整体流程:
1. ./Jocker run /bin/sh
2. 调用/proc/self/exe 即./Jocker init /bin/sh
3. 挂载, 然后syscall.exec执行sh

# Reference
1. [自己动手写docker](https://github.com/xianlubird/mydocker)



