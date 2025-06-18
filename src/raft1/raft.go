package raft

// The file raftapi/raft.go defines the interface that raft must
// expose to servers (or the tester), but see comments below for each
// of these functions for more details.
//
// Make() creates a new raft peer that implements the raft interface.

import (
	"fmt"
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raftapi"
	"6.5840/tester1"
)

// A Go object implementing a single Raft peer.

type Raft struct {
	//工具类以及MIT6824专门需要的
	mu sync.Mutex // Lock to protect shared access to this peer's state
	//cond      *sync.Cond
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *tester.Persister   // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	applyCh   chan raftapi.ApplyMsg
	//所有server维护的需要持久化存储的：
	currentTerm int
	voteFor     int
	log         []LogEntry
	//所有server维护的volatile的
	commitIndex int
	lastApplied int

	//Leader使用到的
	leaderTool LeaderTool

	//其他没有提到但是需要的
	state         string
	leaderID      int       //这个暂时无视掉，第四章可能需要
	lastHeartBeat time.Time //保存上次的心跳时间
	heartBeat     time.Duration

	//更换配置需要用到的
	minimumElectionTimeout time.Duration
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (3A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.state == Leader

	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// how many bytes in Raft's persisted log?
func (rf *Raft) PersistBytes() int {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.persister.RaftStateSize()
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!

// example RequestVote RPC handler.

func (rf *Raft) setNewTermBybiggerTerm(term int) {
	rf.currentTerm = term
	rf.voteFor = -1
	rf.state = Follower
}
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	//term := rf.currentTerm
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		//.Printf("term:%d,serverID:%d---接收到来自server:%d的RequestVote，拒绝:term太低\n", rf.currentTerm, rf.me, args.CandidateID)
		return
	}
	if args.Term > rf.currentTerm {
		rf.setNewTermBybiggerTerm(args.Term)
	}
	//还有一点：我们需要重置心跳时间

	//需要满足1.当前没有投票给其他人或者当前投票的人就是对方，2.日志条目对的上
	qualified := true //检查日志对不对的上，3A先不做
	//
	if (rf.voteFor == -1 || rf.voteFor == args.CandidateID) && qualified {
		reply.VoteGranted = true
		reply.Term = rf.currentTerm
		//fmt.Printf("serverID:%d---接收到来自server:%d的RequestVote，同意\n", rf.me, args.CandidateID)
		//设置心跳
		rf.lastHeartBeat = time.Now()
		rf.voteFor = args.CandidateID //设置candidateID
	} else {
		reply.VoteGranted = false
		reply.Term = rf.currentTerm
		//fmt.Printf("term:%d,serverID:%d---接收到来自server:%d的RequestVote，拒绝:已经投票给别人\n", rf.currentTerm, rf.me, args.CandidateID)
	}
	return
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//fmt.Printf("serverID:%d---接收到来自serverID%d的心跳\n", rf.me, args.LeaderID)
	//reply.Term = rf.currentTerm
	//reply.Success = false
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	//到这里我们基本上保证了Leader没问题
	if args.Term >= rf.currentTerm {
		rf.setNewTermBybiggerTerm(args.Term)
		//并且我们需要更新
	}
	//一致性检查

	//checkResult := true
	//if checkResult == false {
	//	reply.Term = rf.currentTerm
	//	reply.Success = false
	//	return
	//}

	//之后就是通过了一致性检查的
	if len(args.Entries) != 0 {
		//同步条目
	}

	//更新commitIndex
	//if rf.commitIndex > args.LeaderCommit {
	//	rf.commitIndex = min(args.LeaderCommit,最新的日志的Index)
	//}
	rf.lastHeartBeat = time.Now()
	reply.Success = true
	reply.Term = rf.currentTerm
	return
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	if ok {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if reply.Term > rf.currentTerm {
			rf.setNewTermBybiggerTerm(reply.Term)
		}
	}
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	if ok {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if reply.Term > rf.currentTerm {
			rf.setNewTermBybiggerTerm(reply.Term)
		}
	}
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func GetElectionTimeout() int64 {
	return 200 + (rand.Int63() % 301)
}

func (rf *Raft) sendHeartBeat() {
	rf.mu.Lock()
	//rf.lastHeartBeat = time.Now()//student guide说不需要
	//LeaderID := rf.me
	args := AppendEntriesArgs{
		Term:     rf.currentTerm,
		LeaderID: rf.me,

		//设置prevLogIndex和Term,

		//log不用设置

		//设置leaderCommit

	}
	rf.mu.Unlock()
	for serverID, _ := range rf.peers {
		if serverID != rf.me {
			go func(serverID int) {
				reply := AppendEntriesReply{}
				//fmt.Printf("LeaderID:%d---向serverID:%d---发送心跳\n", LeaderID, serverID)
				rf.sendAppendEntries(serverID, &args, &reply)
			}(serverID)
		}
	}
}

func (rf *Raft) ticker() { //心跳机制
	for rf.killed() == false {
		//fmt.Printf("serverID:%d---进入ticker\n", rf.me)
		// Your code here (3A)
		// Check if a leader election should be started.

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		time.Sleep(rf.heartBeat)
		//如果整体上锁，整个操作会具有原子性，但是我们不能在RPC的时候上锁等待回应
		rf.mu.Lock()
		state := rf.state
		rf.mu.Unlock()
		if state == Leader {
			go rf.sendHeartBeat() //应该用线程，否则会阻塞//heartbeat有等幂性，无所谓同时收到多条
		} else {
			electionTimeout := GetElectionTimeout()
			//fmt.Printf("serverID:%d---状态为%s\n", rf.me, rf.state)
			rf.mu.Lock()
			needElection := time.Since(rf.lastHeartBeat).Milliseconds() > electionTimeout
			rf.mu.Unlock()
			//timeout
			if needElection {
				go rf.becomeCandidate() //student guide上面说：即使现在正在选举，我也应该开始新的选举。选举不应该被RPC的延迟而阻塞。
			}
		}
	}
}

//func (rf *Raft) heartBeatLoop() {
//	for {
//		rf.mu.Lock()
//		if rf.state != Leader {
//			rf.mu.Unlock()
//			break
//		}
//		rf.lastHeartBeat = time.Now()
//		//LeaderID := rf.me
//		args := AppendEntriesArgs{
//			Term:     rf.currentTerm,
//			LeaderID: rf.me,
//
//			//设置prevLogIndex和Term,
//
//			//log不用设置
//
//			//设置leaderCommit
//
//		}
//		rf.mu.Unlock()
//
//		//尝试设置Leader降级
//		//serverNum := len(rf.peers)
//		//failCount := 0
//		//sendCount := 0
//		//mu := sync.Mutex{}
//		//wg := sync.WaitGroup{}
//		/////////////////
//		for serverID, _ := range rf.peers {
//			if serverID != rf.me {
//				//wg.Add(1)
//				go func(serverID int) {
//					reply := AppendEntriesReply{}
//					//fmt.Printf("LeaderID:%d---向serverID:%d---发送心跳\n", LeaderID, serverID)
//					rf.sendAppendEntries(serverID, &args, &reply)
//					////降级相关
//					//mu.Lock()
//					//sendCount++
//					//if ok == false {
//					//	failCount++
//					//}
//					//mu.Unlock()
//					//wg.Done()
//				}(serverID)
//			}
//		}
//		//wg.Wait()
//		//if failCount > serverNum/2 {
//		//	rf.mu.Lock()
//		//	rf.state = Follower
//		//	rf.voteFor = -1
//		//	rf.mu.Unlock()
//		//}
//
//		time.Sleep(rf.heartBeat) //等待下一次心跳
//	}
//}

func (rf *Raft) becomeLeader() {
	rf.mu.Lock()

	rf.state = Leader
	//后续还需要设置nextIndex以及matchIndex
	rf.mu.Unlock()
	//go rf.heartBeatLoop()
	go rf.sendHeartBeat() //上任需要发送一次心跳
	//发送no-op命令
}

func (rf *Raft) becomeCandidate() {
	rf.mu.Lock()

	rf.state = Candidate
	rf.currentTerm++
	rf.voteFor = rf.me
	rf.lastHeartBeat = time.Now()
	args := RequestVoteArgs{
		Term:        rf.currentTerm,
		CandidateID: rf.me,
		//LastLogIndex:0
		//LastLogTerm:0
	}
	fmt.Printf("term:%d,serverID:%d---成为候选者\n", args.Term, rf.me)
	rf.mu.Unlock()
	//向每个节点请求投票
	mu := sync.Mutex{}
	cond := sync.NewCond(&mu)
	wg := sync.WaitGroup{}
	grantedCount := 1
	receivedCount := 1
	serverNum := len(rf.peers)
	//fmt.Printf("当前服务器的总数为:%d\n", len(rf.peers))
	for serverID, _ := range rf.peers {
		if serverID != rf.me {
			wg.Add(1)
			go func(serverID int) {
				//为每条请求设置单独的reply
				reply := RequestVoteReply{}
				ok := rf.sendRequestVote(serverID, &args, &reply)
				mu.Lock()
				//rpc请求可能传输不到，这种情况下我们也默认传输到了，算作被拒绝
				receivedCount++
				if ok && reply.VoteGranted {
					grantedCount++
					fmt.Printf("term:%d,serverID:%d---收到了来自serverID:%d,对方同意了投票\n", args.Term, rf.me, serverID)
				}
				mu.Unlock()
				wg.Done()
				cond.Broadcast()
			}(serverID)
		}
	}
	mu.Lock()
	//思考：可能会一直被阻塞吗？
	for grantedCount <= serverNum/2 && receivedCount < serverNum {
		cond.Wait()
	}
	//fmt.Printf("term:%d,serverID:%d---获得的总票数为:%d\n", rf.currentTerm, rf.me, grantedCount)
	if grantedCount > serverNum/2 {
		fmt.Printf("term:%d,serverID:%d---成为Leader\n", rf.currentTerm, rf.me)
		go rf.becomeLeader()
	}
	mu.Unlock()
	wg.Wait()
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.

//type Raft struct {
//	//MIT6824专门需要的

//	mu        sync.Mutex          // Lock to protect shared access to this peer's state
//  //cond	  *sync.Cond
//	peers     []*labrpc.ClientEnd // RPC end points of all peers
//	persister *tester.Persister   // Object to hold this peer's persisted state
//	me        int                 // this peer's index into peers[]
//	dead      int32               // set by Kill()
//	applyCh chan raftapi.ApplyMsg

//	//所有server维护的需要持久化存储的：
//	currentTerm int
//	voteFor     int
//	log         []LogEntry
//	//所有server维护的volatile的
//	commitIndex int
//	lastApplied int
//
//	//Leader使用到的
//	leaderTool LeaderTool
//

//}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *tester.Persister, applyCh chan raftapi.ApplyMsg) raftapi.Raft {
	rf := &Raft{}
	//工具类的设置
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applyCh = applyCh
	//rf.cond = sync.NewCond(&rf.mu)
	//持久化数据
	rf.currentTerm = 0
	rf.voteFor = -1
	rf.log = make([]LogEntry, 0)
	//非持久化数据
	rf.commitIndex = -1
	rf.lastApplied = -1
	//Leader相关数据
	rf.leaderTool = LeaderTool{} //选上了再初始化
	//一些没有提到但是需要的
	rf.state = Follower
	rf.leaderID = -1
	rf.lastHeartBeat = time.Now()
	//心跳的时间间隔：
	rf.heartBeat = time.Duration(50) * time.Millisecond
	rf.minimumElectionTimeout = 100 * time.Millisecond
	//fmt.Printf("serverID:%d---创建\n", me)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker() //初始化完毕，投入工作

	return rf
}
