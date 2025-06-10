package mr

import (
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

type Task struct {
	FileName  string
	TaskID    int
	Status    string
	StartTime time.Time
}
type Coordinator struct {
	// Your definitions here.
	//同步机制
	mu sync.Mutex
	//记录两种任务
	MapTasks    []Task
	ReduceTasks []Task
	//记录nreduce也就是reduce阶段的任务数量
	NReduce int
	//记录两阶段任务完成情况
	MapFinishied bool
	AllFinished  bool
}

// Your code here -- RPC handlers for the worker to call.

//type GetTaskReply struct {
//	FileName   string
//	TaskID     int
//	TaskType   string
//	NReduce    int
//	MapTaskNum int
//}

func (c *Coordinator) GetTask(args *GetTaskArgs, reply *GetTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	//
	c.checkTimeout() //检查超时
	if c.AllFinished {
		reply.TaskType = ExitTask
		return nil
	}
	if !c.MapFinishied {
		//fmt.Printf("这一次我们执行的是map阶段的函数\n")
		for i, task := range c.MapTasks {
			if task.Status == Idle {
				reply.TaskType = MapTask
				reply.TaskID = task.TaskID
				reply.NReduce = c.NReduce
				reply.FileName = task.FileName
				c.MapTasks[i].Status = Progressing
				c.MapTasks[i].StartTime = time.Now()
				return nil //如果能找到空闲的,那么就可以直接返回了
			}
		}
		//如果完成了遍历之后还是没有找到未完成的任务
		reply.TaskType = WaitTask
		return nil
	}
	for i, task := range c.ReduceTasks { //不满足上面这个条件，那么必然就是处于reduce阶段
		//switch task.Status {
		//case Idle:
		//	fmt.Printf("当前任务%d处于空闲状态\n", i)
		//case Progressing:
		//	fmt.Printf("当前任务%d处于执行状态\n", i)
		//case Done:
		//	fmt.Printf("当前任务%d处于完成状态\n", i)
		//}
		if task.Status == Idle {
			//fmt.Printf("执行到了reduceTask内部")
			reply.TaskType = ReduceTask
			reply.TaskID = task.TaskID
			reply.NReduce = c.NReduce
			reply.FileName = task.FileName //个人感觉fileName是无用的
			reply.MapTaskNum = len(c.MapTasks)
			c.ReduceTasks[i].Status = Progressing
			c.ReduceTasks[i].StartTime = time.Now()
			return nil
		}

	}
	reply.TaskType = WaitTask
	return nil
}

func (c *Coordinator) checkTimeout() {
	if c.AllFinished {
		return
	}
	timeout := 10 * time.Second
	now := time.Now()
	if !c.MapFinishied {
		completed := true
		for i, task := range c.MapTasks {
			if task.Status == Progressing && now.Sub(task.StartTime) > timeout {
				c.MapTasks[i].Status = Idle
			}
			if task.Status != Done {
				completed = false
			}
		}
		//if completed {
		//	fmt.Printf("将mapFinishied设置为true\n")
		//} else {
		//	fmt.Printf("将mapFinishied设置为false\n")
		//}
		c.MapFinishied = completed
		return
	}
	completed := true
	for i, task := range c.ReduceTasks {
		//fmt.Printf("检查reducetask%d\n", i)
		if task.Status == Progressing && now.Sub(task.StartTime) > timeout {
			c.ReduceTasks[i].Status = Idle
		}
		if task.Status != Done {
			completed = false
		}
	}
	//if completed {
	//	fmt.Printf("将AllFinishied设置为true\n")
	//} else {
	//	fmt.Printf("将AllFinishied设置为false\n")
	//}
	c.AllFinished = completed
	return
}

func (c *Coordinator) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if args.TaskType == MapTask {
		for i, task := range c.MapTasks {

			if task.Status == Progressing && task.TaskID == args.TaskID {

				c.MapTasks[i].Status = Done
				completed := true
				for _, task := range c.MapTasks {
					if task.Status != Done {
						completed = false
						break
					}
				}
				//if completed {
				//	fmt.Printf("将mapFinishied设置为true\n")
				//} else {
				//	fmt.Printf("将mapFinishied设置为false\n")
				//}
				c.MapFinishied = completed
				reply.Ok = true
				return nil
			}
		}
	} else if args.TaskType == ReduceTask {
		for i, task := range c.ReduceTasks {
			//fmt.Printf("检查reducetask%d\n", i)
			if task.Status == Progressing && task.TaskID == args.TaskID {
				//fmt.Printf("发生了一次将reduceTask设置为Done状态\n")
				c.ReduceTasks[i].Status = Done
				completed := true
				for _, task := range c.ReduceTasks {
					if task.Status != Done {
						completed = false
						break
					}
				}
				//if completed {
				//	fmt.Printf("将AllFinishied设置为true\n")
				//} else {
				//	fmt.Printf("将AllFinishied设置为false\n")
				//}
				c.AllFinished = completed
				reply.Ok = true
				return nil
			}
		}
	}
	reply.Ok = false
	return nil
}

func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false
	if c.AllFinished {
		ret = true
	}
	// Your code here.
	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	//mu 	sync.Mutex
	////记录两种任务
	//MapTasks    []Task
	//ReduceTasks []Task
	////记录nreduce也就是reduce阶段的任务数量
	//NReduce 	int
	////记录两阶段任务完成情况
	//MapFinishied bool
	//AllFinished bool
	c := Coordinator{
		MapTasks:     make([]Task, len(files)),
		ReduceTasks:  make([]Task, nReduce),
		NReduce:      nReduce,
		MapFinishied: false,
		AllFinished:  false,
	}
	//初始化map任务
	/*	FileName  string
		TaskID    int
		Status    string
		StartTime time.Time*/
	for i, filename := range files {
		c.MapTasks[i] = Task{
			FileName: filename,
			TaskID:   i,
			Status:   Idle,
		}
	}
	for i := 0; i < c.NReduce; i++ {
		c.ReduceTasks[i] = Task{
			TaskID: i,
			Status: Idle,
		}
		//开始时间和文件名都不定义，文件名可以通过ID来推导
	}

	c.server()
	return &c
}
