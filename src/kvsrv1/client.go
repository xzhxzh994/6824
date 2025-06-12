package kvsrv

import (
	"6.5840/kvsrv1/rpc"
	"6.5840/kvtest1"
	"6.5840/tester1"
	"fmt"
	"time"
)

type Clerk struct {
	clnt   *tester.Clnt
	server string
}

func MakeClerk(clnt *tester.Clnt, server string) kvtest.IKVClerk {
	ck := &Clerk{clnt: clnt, server: server}
	// You may add code here.
	return ck
}

// Get fetches the current value and version for a key.  It returns
// ErrNoKey if the key does not exist. It keeps trying forever in the
// face of all other errors.
//
// You can send an RPC with code like this:
// ok := ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
// Get 获取指定键的当前值和版本。如果键不存在，它将返回 ErrNoKey。
// 在面对其他所有错误时，它会无限重试。
//
// 你可以用以下代码发送一个RPC请求：
// ok := ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply)
//
// args 和 reply 的类型（包括是否是指针）必须与RPC处理函数的
// 参数声明类型相匹配。此外，reply 必须作为指针传递。
//	type GetArgs struct {
//		Key string
//	}
//
//	type GetReply struct {
//		Value   string
//		Version Tversion
//		Err     Err
//	}

func (ck *Clerk) Get(key string) (string, rpc.Tversion, rpc.Err) {
	// You will have to modify this function.
	args := rpc.GetArgs{Key: key}
	reply := rpc.GetReply{}
	ok := ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply)
	for ok == false {
		time.Sleep(100 * time.Millisecond)
		//fmt.Printf("尝试重发Get")
		ok = ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply)

	} //不断尝试call

	//ok := ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply)
	//if ok == false {
	//	log.Fatalf("can not get key %s\n", key)
	//}
	//if reply.Err != rpc.OK {
	//	return "", 0, reply.Err
	//}
	return reply.Value, reply.Version, reply.Err

}

// Put updates key with value only if the version in the
// request matches the version of the key at the server.  If the
// versions numbers don't match, the server should return
// ErrVersion.  If Put receives an ErrVersion on its first RPC, Put
// should return ErrVersion, since the Put was definitely not
// performed at the server. If the server returns ErrVersion on a
// resend RPC, then Put must return ErrMaybe to the application, since
// its earlier RPC might have been processed by the server successfully
// but the response was lost, and the Clerk doesn't know if
// the Put was performed or not.
//
// You can send an RPC with code like this:
// ok := ck.clnt.Call(ck.server, "KVServer.Put", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
// Put 更新键值，仅当请求中的版本与服务器上该键的版本匹配时才会更新。
// 如果版本号不匹配，服务器应返回 ErrVersion。
// 如果 Put 在第一次 RPC 调用时接收到 ErrVersion，Put 应该返回 ErrVersion，
// 因为该 Put 操作肯定没有在服务器上执行。
// 如果服务器在重新发送的 RPC 中返回 ErrVersion，
// 那么 Put 必须返回 ErrMaybe 给应用程序，因为它之前的 RPC 可能已经被服务器成功处理，
// 但响应丢失了，Clerk 无法知道该 Put 操作是否被执行。
//
// 你可以用以下代码发送一个 RPC 请求：
// ok := ck.clnt.Call(ck.server, "KVServer.Put", &args, &reply)
//
// args 和 reply 的类型（包括是否是指针）必须与 RPC 处理函数的
// 参数声明类型相匹配。此外，reply 必须作为指针传递。

func (ck *Clerk) Put(key, value string, version rpc.Tversion) rpc.Err {
	// You will have to modify this function.
	args := rpc.PutArgs{Key: key, Value: value, Version: version}
	reply := rpc.PutReply{}
	//
	ok := ck.clnt.Call(ck.server, "KVServer.Put", &args, &reply)
	if ok == true {
		return reply.Err
	}
	for ok == false {
		time.Sleep(100 * time.Millisecond)
		ok = ck.clnt.Call(ck.server, "KVServer.Put", &args, &reply)
		//fmt.Printf("尝试重发Put")
	}
	if reply.Err == rpc.ErrVersion {
		return rpc.ErrMaybe
	} else {
		return reply.Err
	}

}
