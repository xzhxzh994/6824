package lock

import (
	"log"
	"time"

	"6.5840/kvsrv1/rpc"
	"6.5840/kvtest1"
)

const (
	Unlocked = "unlocked"
)

type Lock struct {
	// IKVClerk is a go interface for k/v clerks: the interface hides
	// the specific Clerk type of ck but promises that ck supports
	// Put and Get.  The tester passes the clerk in when calling
	// MakeLock().
	// IKVClerk 是一个用于键值（k/v）客户端的 Go 接口：这个接口隐藏了具体的 Clerk 类型，但保证了该 Clerk 支持 Put 和 Get 操作。
	// 测试器在调用 MakeLock() 时会传递该 Clerk。
	ck       kvtest.IKVClerk //
	ClientID string
	LockKey  string
	//Lease lease

	// You may add code here
}

// The tester calls MakeLock() and passes in a k/v clerk; your code can
// perform a Put or Get by calling lk.ck.Put() or lk.ck.Get().
//
// Use l as the key to store the "lock state" (you would have to decide
// precisely what the lock state is).
// 测试器调用 MakeLock() 并传入一个 k/v 客户端；你的代码可以通过调用 lk.ck.Put() 或 lk.ck.Get() 来执行 Put 或 Get 操作。
//
// 使用 l 作为键来存储 "锁的状态"（你需要决定具体的锁状态是什么）。
func MakeLock(ck kvtest.IKVClerk, l string) *Lock {
	//client传入想要访问的服务器，以及一个string l来唯一标识这个lock
	lk := &Lock{
		ck:       ck,
		ClientID: kvtest.RandValue(8),
		LockKey:  l,
	}
	// You may add code here
	//kv服务器中有一个l保存着锁的状态，我们makeLock的时候应该检查这个l是否存在，如果不存在，那么我们需要创建这个l
	_, _, err := lk.ck.Get(l)
	if err == rpc.ErrNoKey {
		lk.ck.Put(l, Unlocked, 0)
	}
	return lk
}

//Get(string) (string, rpc.Tversion, rpc.Err)
//Put(string, string, rpc.Tversion) rpc.Err

//这里我们定义KV服务器上维护一个标志位，这个标志的值在
//未上锁的时候：Unlocked
//上锁的时候：对应的上锁的ID

func (lk *Lock) Acquire() {
	// Your code here
	for {
		state, version, err := lk.ck.Get(lk.LockKey)
		if err == rpc.ErrNoKey {
			log.Fatalf("can not find lock key %s", lk.LockKey)
		}
		if state == Unlocked || state == lk.ClientID { //如果处于未锁定状态
			lk.ck.Put(lk.LockKey, lk.ClientID, version)
			break
		} else {
			time.Sleep(time.Second) //谁一秒在继续for循环
		}
	}

	for {
		state, version, err := lk.ck.Get(lk.LockKey)
		if err == rpc.ErrNoKey {
			log.Fatalf("can not find lock key %s", lk.LockKey)
		}
		if state == Unlocked || state == lk.ClientID { //说明我们可以继续操作
			puterr := lk.ck.Put(lk.LockKey, lk.ClientID, version) //这是一个竞态安全操作
			if puterr == rpc.OK {                                 //说明我们的put成功了
				break
			} //否则我们继续下一次的尝试
		}
		time.Sleep(50 * time.Millisecond)
	}

}

func (lk *Lock) Release() {
	// Your code here
	state, version, err := lk.ck.Get(lk.LockKey)
	if err == rpc.ErrNoKey {
		log.Fatalf("can not find lock key %s", lk.LockKey)
	}
	if state == lk.ClientID { //如果就是我们这个ID上的锁
		lk.ck.Put(lk.LockKey, Unlocked, version)
	} else {
		log.Fatalf("somehow lock is not belong to this client %s", lk.ClientID)
	}

}
