package mr

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	workerID := os.Getpid()
	for {
		//首先请求任务
		task := getTask(workerID)
		switch task.TaskType {
		case MapTask:
			doMap(task, mapf, workerID)
		case ReduceTask:
			doReduce(task, reducef, workerID)
		case WaitTask:
			time.Sleep(1 * time.Second)
		case ExitTask:
			return
		}
	}
	//worker是一个进程，它不断向coordinator请求任务，执行，完成之后报告
}

func doMap(task GetTaskReply, mapf func(string, string) []KeyValue, workerID int) {
	filename := task.FileName
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("can not open file %v", err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("can not read file %v", err)
	}
	intermediate := make([][]KeyValue, task.NReduce)
	kva := mapf(filename, string(content))
	for _, kv := range kva {
		hash := ihash(kv.Key) % task.NReduce
		intermediate[hash] = append(intermediate[hash], kv)
	}
	//将每个桶的文件放到持久化存储

	for i := 0; i < task.NReduce; i++ {
		tempfile, err := os.CreateTemp("", "mr-tmp-*")
		if err != nil {
			log.Fatalf("can not create tempfile %v", err)
		}
		enc := json.NewEncoder(tempfile)
		for _, kv := range intermediate[i] {
			err := enc.Encode(kv)
			if err != nil {
				log.Fatalf("can not encode %v", err)
			}
		}
		tempfile.Close()
		os.Rename(tempfile.Name(), fmt.Sprintf("mr-%d-%d", task.TaskID, i))
	}
	reportTask(task.TaskType, workerID, task.TaskID)
}

func doReduce(task GetTaskReply, reducef func(string, []string) string, workerID int) {
	maptasknum := task.MapTaskNum
	taskID := task.TaskID

	intermediate := []KeyValue{}

	for i := 0; i < maptasknum; i++ {
		filename := fmt.Sprintf("mr-%d-%d", i, taskID)
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("can not open file %v", err)
		}
		dec := json.NewDecoder(file)
		for {
			var kva KeyValue
			if err := dec.Decode(&kva); err != nil {
				break
			}
			intermediate = append(intermediate, kva)
		}
		file.Close()
	}
	//至此我们读完了所有的mr-i-taskID文件内的内容
	//shuffle
	sort.Sort(ByKey(intermediate))
	tempfile, err := os.CreateTemp("", "mr-out-tmp-*")
	if err != nil {
		log.Fatalf("can not create tempfile %v", err)
	}
	//之后我们要把这其中的内容写到对应的文件中
	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)
		fmt.Fprintf(tempfile, "%v %v\n", intermediate[i].Key, output)
		i = j
	}
	tempfile.Close()
	os.Rename(tempfile.Name(), fmt.Sprintf("mr-out-%d", taskID))、
	reportTask(task.TaskType, workerID, task.TaskID)

}

func getTask(workerID int) GetTaskReply {
	args := GetTaskArgs{
		WokerID: workerID,
	}
	reply := GetTaskReply{}

	call("coordinator.GetTask", &args, &reply)
	return reply
}

func reportTask(tasktype string, workerID int, taskID int) {
	args := ReportTaskArgs{
		TaskType: tasktype,
		WorkerID: workerID,
		TaskID:   taskID,
	}
	reply := ReportTaskReply{}
	call("coordinator.ReportTask", &args, &reply)
}

// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
