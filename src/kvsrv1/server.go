package kvsrv

import (
	"log"
	"sync"

	"6.5840/kvsrv1/rpc"
	"6.5840/labrpc"
	"6.5840/tester1"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type ValueVersion struct {
	Value   string
	Version rpc.Tversion
}
type KVServer struct {
	mu    sync.Mutex
	Kvmap map[string]ValueVersion
	// Your definitions here.

}

func MakeKVServer() *KVServer {
	kv := &KVServer{
		Kvmap: make(map[string]ValueVersion),
	}
	// Your code here.
	//kv.server()
	return kv
}

// Get returns the value and version for args.Key, if args.Key
// exists. Otherwise, Get returns ErrNoKey.

//	type GetArgs struct {
//		Key string
//	}
//
//	type GetReply struct {
//		Value   string
//		Version Tversion
//		Err     Err
//	}

func (kv *KVServer) Get(args *rpc.GetArgs, reply *rpc.GetReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	Key := args.Key
	VV, exists := kv.Kvmap[Key]
	if !exists {
		reply.Value = ""
		reply.Version = 0
		reply.Err = rpc.ErrNoKey
		return
	}
	reply.Version = VV.Version
	reply.Value = VV.Value
	reply.Err = rpc.OK
}

// Update the value for a key if args.Version matches the version of
// the key on the server. If versions don't match, return ErrVersion.
// If the key doesn't exist, Put installs the value if the
// args.Version is 0, and returns ErrNoKey otherwise.
//
//	type PutArgs struct {
//		Key     string
//		Value   string
//		Version Tversion
//	}
//
//	type PutReply struct {
//		Err Err
//	}
func (kv *KVServer) Put(args *rpc.PutArgs, reply *rpc.PutReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	VV, exists := kv.Kvmap[args.Key]
	if !exists { //不存在这个key
		if args.Version != 0 {
			reply.Err = rpc.ErrNoKey
			return
		}
		//fmt.Printf("虽然不存在这个键，但是我将会创建这个key\n")
		kv.Kvmap[args.Key] = ValueVersion{Value: args.Value, Version: args.Version + 1}
		reply.Err = rpc.OK
	} else { //存在这个key
		//判断版本对不对
		if args.Version != VV.Version {
			reply.Err = rpc.ErrVersion
			return
		}
		//如果版本一致，则赋值过去
		kv.Kvmap[args.Key] = ValueVersion{Value: args.Value, Version: args.Version + 1}
		reply.Err = rpc.OK
	}

}

// You can ignore Kill() for this lab
func (kv *KVServer) Kill() {
}

// You can ignore all arguments; they are for replicated KVservers
func StartKVServer(ends []*labrpc.ClientEnd, gid tester.Tgid, srv int, persister *tester.Persister) []tester.IService {
	kv := MakeKVServer()
	return []tester.IService{kv}
}

//func (s *KVServer) server() {
//	rpc_.Register(s)
//	rpc_.HandleHTTP()
//	//l, e := net.Listen("tcp", ":1234")
//	sockname := rpc.ServerSock()
//	os.Remove(sockname)
//	l, e := net.Listen("unix", sockname)
//	if e != nil {
//		log.Fatal("listen error:", e)
//	}
//	go http.Serve(l, nil)
//}
