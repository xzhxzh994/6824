package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//定义一些常见的状态

// 1.任务执行的状态
const (
	Idle        = "idle"        //未开始
	Progressing = "progressing" //执行中
	Done        = "done"        //已完成
)

// 任务类型
const (
	MapTask    = "map"
	ReduceTask = "reduce"
	WaitTask   = "wait"
	ExitTask   = "exit"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type GetTaskArgs struct {
	WokerID int
}

type GetTaskReply struct {
	FileName   string
	TaskID     int
	TaskType   string
	NReduce    int
	MapTaskNum int
}

type ReportTaskArgs struct {
	WorkerID int
	TaskType string
	TaskID   int
}
type ReportTaskReply struct {
	Ok bool
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
