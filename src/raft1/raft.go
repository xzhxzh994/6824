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
	tester "6.5840/tester1"
)

// A Go object implementing a single Raft peer.

type Raft struct {
	//工具类以及MIT6824专门需要的
	mu        sync.Mutex // Lock to protect shared access to this peer's state
	applyCond *sync.Cond
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *tester.Persister   // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	applyCh   chan raftapi.ApplyMsg
	//所有server维护的需要持久化存储的：
	currentTerm int
	voteFor     int
	log         []LogEntry
	//log Log
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
		fmt.Printf("term:%d,serverID:%d---接收到来自server:%d的RequestVote，拒绝:term太低\n", rf.currentTerm, rf.me, args.CandidateID)
		return
	}
	if args.Term > rf.currentTerm {
		rf.setNewTermBybiggerTerm(args.Term)
	}

	//需要满足1.当前没有投票给其他人或者当前投票的人就是对方，2.日志条目跟自己的一样新

	qualified := rf.log[len(rf.log)-1].Term < args.LastLogTerm || (rf.log[len(rf.log)-1].Term == args.LastLogTerm && rf.log[len(rf.log)-1].Index <= args.LastLogIndex) //检查日志对不对的上，3A先不做
	if qualified == true {
		fmt.Printf("term:%d,serverID:%d---接收到来自server:%d的RequestVote,日志条目跟自己的一样新,对方的lastLogIndex = %d,lastLogTerm = %d,自己的lastLogIndex = %d,lastLogTerm = %d\n", rf.currentTerm, rf.me, args.CandidateID, args.LastLogIndex, args.LastLogTerm, rf.log[len(rf.log)-1].Index, rf.log[len(rf.log)-1].Term)
	}

	//
	//qualified := true
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
		if qualified == false {
			fmt.Printf("term:%d,serverID:%d---接收到来自server:%d的RequestVote，拒绝:日志条目落后于自己\n", rf.currentTerm, rf.me, args.CandidateID)
		} else {
			fmt.Printf("term:%d,serverID:%d---接收到来自server:%d的RequestVote，拒绝:已经投票给别人\n", rf.currentTerm, rf.me, args.CandidateID)

		}

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
	rf.lastHeartBeat = time.Now() //此时我们应该立刻更新心跳，否则后面会有false的地方，导致心跳更新不到位
	//一致性检查
	//检查是否存在对应索引的对应Term
	checkResult := false
	pos := 0

	//如果当前的目录是空的，我们应该接受Leader发过来的日志///不存在这种情况
	//如果Leader发来的日志的prevLogIndex是//也不存在这种情况
	//if len(rf.log) == 0 ||  {
	//	checkResult = true
	//}
	//先从后往前找到第一个符合term的
	for i := len(rf.log) - 1; i >= 0; i-- {
		if rf.log[i].Index == args.PrevLogIndex && rf.log[i].Term == args.PrevLogTerm {
			checkResult = true
			pos = i
			break
		}
		if rf.log[i].Index < args.PrevLogIndex { //当遍历到索引更小的位置的时候就不用遍历了，因为已经不存在了。
			break
		}
	}
	//
	if checkResult == false {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	} //通过一致性检查

	//之后就是通过了一致性检查的
	if len(args.Entries) != 0 {
		//同步条目
		//pos是prevLog
		for pos, i := pos+1, 0; i < len(args.Entries); i, pos = i+1, pos+1 {
			//如果日志的最后一条也被我们检查了,说明接下来的所有Entries都应该被直接append到末尾
			if pos >= len(rf.log) { //由于我们保证了每个日志中最少有一个空的条目,pos最坏情况下为1
				rf.log = append(rf.log, args.Entries[i:]...)
				break
			}
			//如果pos位置有的话，我们检查它们是否相等
			if rf.log[pos].Index != args.Entries[i].Index || rf.log[pos].Term != args.Entries[i].Term { //如果存在某一条不一样，包括这一条后面的全部都被覆盖掉
				rf.log = append(rf.log[:pos], args.Entries[i:]...)
				break
			}
		}
		//更新commitIndex,只有在更新过日志之后才能更新commitIndex

	}
	
	// if rf.commitIndex < args.LeaderCommit && rf.commitIndex < args.PrevLogIndex {
		
	// 	rf.commitIndex = min(args.LeaderCommit, args.PrevLogIndex) //prevLogIndex表示了和Leader一致的最新一条索引
	// 	fmt.Printf("Term:%d,serverID:%d---接收到来自server:%d的AppendEntries,更新commitIndex为%d\n", rf.currentTerm, rf.me, args.LeaderID, args.PrevLogIndex)
	// 	rf.apply()
	// }
	if rf.commitIndex < args.LeaderCommit {
		commitINdex := rf.commitIndex
		rf.commitIndex = min(args.LeaderCommit, rf.log[len(rf.log)-1].Index) //prevLogIndex表示了和Leader一致的最新一条索引\
		if commitIndex != rf.commitIndex {
			fmt.Printf("Term:%d,serverID:%d---接收到来自server:%d的AppendEntries,尝试更新commitIndex为%d\n", rf.currentTerm, rf.me, args.LeaderID, args.LeaderCommit)
			
		}
		rf.apply()
	}

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
	//可能得到的结果:
	//1.reply.success== true,说明对方收到了并且认可，可能还复制了日志
	//2.reply.success== false && reply.term>currentTerm 说明对方不认可自己作为Leader
	//3.reply.success== false && reply.term==currentTerm
	//if ok {
	//	if reply.Success == false {
	//		rf.mu.Lock()
	//		defer rf.mu.Unlock()
	//		if reply.Term > rf.currentTerm {
	//			rf.setNewTermBybiggerTerm(reply.Term)
	//		} else if reply.Term == rf.currentTerm { //说明我们一致性检查没通过
	//			rf.leaderTool.nextIndex[server]--
	//			//只有在发送条目的情况下，我们才使用协程，思考：这里的reply作为一个参数，会不会由于当前协程的提前返回而出现问题
	//			//go rf.sendAppendEntries(server, args, reply) //重试，为防止阻塞，这里设置成独立线程
	//		}
	//	} else {
	//
	//	}
	//}
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
// 使用Raft的服务（例如一个键值存储服务器）希望启动
// 下一条命令的共识协议，该命令将被追加到Raft的日志中。如果此
// 服务器不是Leader，返回false。否则，立即启动共识
// 并返回。不能保证该命令最终会被提交到Raft日志中，因为
// Leader可能会失败或丧失选举资格。即使Raft实例已被终止，
// 该函数也应该优雅地返回。
//
// 第一个返回值是该命令如果最终被提交时，出现的索引。
// 第二个返回值是当前的任期。
// 第三个返回值是如果此服务器认为自己是Leader，则返回true。
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	fmt.Printf("Term = %d,node = %d,Start中的AppendEntries获得锁\n", rf.currentTerm, rf.me)

	defer fmt.Printf("Term = %d,node = %d,Start中的AppendEntries释放锁\n", rf.currentTerm, rf.me)
	defer rf.mu.Unlock()
	index := -1
	term := -1
	isLeader := false

	if rf.killed() || rf.state != Leader {
		fmt.Printf("Term = %d,Not Leader = %d,尝试start条目，失败\n", rf.currentTerm, rf.me)
		return index, term, isLeader
	}
	//当前节点就是Leader并且是活跃的
	//发送过来的Index和Term必须先存到Leader上，不然可能会导致Index乱序。
	isLeader = true
	term = rf.currentTerm
	index = len(rf.log) //对于尝试数组，应该添加的是Index = 1,log拥有一条默认条目，所以len = 1
	fmt.Printf("Term = %d,Leader = %d,尝试start条目:%d,term = %d\n", rf.currentTerm, rf.me, index, term)
	rf.log = append(rf.log, LogEntry{
		Term:    term,
		Command: command,
		Index:   index,
	})
	go rf.tryToCommit(term, index) //

	return index, term, isLeader
}

// tryCommit只能被Start调用，Start调用时，此条目一定是当前的term
func (rf *Raft) tryToCommit(term int, index int) {
	//将这条Command放到自己的目录中，然后通过 AppendEntries 复制到别的日志中。(一直重复，直到成功或者失败为止)
	//之后更新commitIndex/nextIndex/matchIndex

	//将日志复制到其他节点

	//我们需要一个Counter来记录AppendEntries成功的数量
	Counter := 1 //先算上自己
	//rf.mu.Lock()//这里会导致死锁
	currenterm := rf.currentTerm
	//rf.mu.Unlock()
	mu := sync.Mutex{}
	cond := sync.NewCond(&mu)
	//wg := sync.WaitGroup{}
	currentTerm := rf.currentTerm

	//对每个节点：发送AppendEntries，附上从nextIndex到最新日志的log。如果回复false则应该重发。
	//但是这里考虑一个情况:由于网络分区,Leader已经不再是真的Leader，那么重发就永远不会得到认可。所以我们要检查false的原因，如果是一致性检查未通过或者ok = false，则可以重发，如果是Term小于reply，则停止
	for serverID, _ := range rf.peers {
		if serverID != rf.me {
			//wg.Add(1)
			go func(serverID int, currentTerm int) {
				//defer wg.Done()
				//只有  当前term  的条目保存到了majority，成为了可提交状态。在这个之前的条目才会被赋予committed属性 作为可提交状态。
				for {
					args := AppendEntriesArgs{
						Term:         currenterm,
						LeaderID:     rf.me,
						LeaderCommit: rf.commitIndex, //不用担心，只要是commit的一定是安全的
						PrevLogIndex: rf.leaderTool.nextIndex[serverID] - 1,
						PrevLogTerm:  rf.log[rf.leaderTool.nextIndex[serverID]-1].Term,
					}
					args.Entries = rf.log[args.PrevLogIndex+1:]
					reply := AppendEntriesReply{}
					ok := rf.sendAppendEntries(serverID, &args, &reply)
					//如果发送成功并且添加成功
					//rf.mu.Lock()
					//fmt.Printf("Term = %d,Leader = %d,trycommit中的AppendEntries获得锁\n", rf.currentTerm, rf.me)
					//defer fmt.Printf("Term = %d,Leader = %d,trycommit中的AppendEntries释放锁\n", rf.currentTerm, rf.me)
					//defer rf.mu.Unlock()
					if reply.Success == true {
						mu.Lock()
						fmt.Printf("Term = %d,Leader = %d,收到来自节点= %d的 添加成功响应\n", rf.currentTerm, rf.me, serverID)
						Counter++
						defer mu.Unlock()
						rf.leaderTool.nextIndex[serverID] = max(rf.leaderTool.nextIndex[serverID], index+1)
						rf.leaderTool.matchIndex[serverID] = max(rf.leaderTool.matchIndex[serverID], index)
						cond.Broadcast()
						break
					}
					if ok == false { //没收到必须无限重发
						continue
					}
					//如果发送成功，但是添加不成功

					if reply.Term > rf.currentTerm { //不是真正的Leader

						rf.setNewTermBybiggerTerm(reply.Term)

						cond.Broadcast() //唤醒一次，
						break
					} //没有通过一致性检查
					rf.leaderTool.nextIndex[serverID] = max(rf.leaderTool.nextIndex[serverID]-1, 1)
				}

			}(serverID, currentTerm)
		}
	}
	mu.Lock()
	//只有这条消息得到了超过半数的同意，或者我不是Leader了（因为term落后）
	for Counter <= len(rf.peers)/2 && rf.state == Leader {
		cond.Wait()
	}
	mu.Unlock()
	rf.mu.Lock()
	fmt.Printf("Term = %d,Leader = %d,trycommit中获得锁\n", rf.currentTerm, rf.me)
	defer fmt.Printf("Term = %d,Leader = %d,trycommit中释放锁\n", rf.currentTerm, rf.me)
	defer rf.mu.Unlock()
	fmt.Printf("Term = %d,Leader = %d,退出等待状态，条目复制到majority或者不是Leader\n", rf.currentTerm, rf.me)
	if Counter > len(rf.peers)/2 && term == rf.currentTerm { //超过半数的人完成了append,并且这个条目是当前任期的
		fmt.Printf("Leader = %d,成功commit条目%d \n", rf.me, index)
		rf.commitIndex = max(rf.commitIndex, index)
		rf.apply()
	} else if Counter <= len(rf.peers)/2 {
		fmt.Printf("Term = %d,Leader = %d,无法执行commitIndex的更新,原因:票数不足\n", rf.currentTerm, rf.me)
	} else {
		fmt.Printf("Term = %d,Leader = %d,无法执行commitIndex的更新,原因:当前任期不对,term = %d,rf.currentTerm = %d\n", rf.currentTerm, rf.me, term, rf.currentTerm)
	}
	//自己不是leader了应该就不用管了
	//wg.Wait()
	fmt.Printf("term = %d,Leader = %d,commit执行完成\n", rf.currentTerm, rf.me)

}

//func (rf *Raft) tryToAppend(serverID int) {
//	rf.mu.Lock()
//	prevLogIndex, prevLogTerm, pos := rf.getPrev(serverID)
//	args := AppendEntriesArgs{
//		Term:         rf.currentTerm,
//		LeaderID:     rf.me,
//		PrevLogIndex: prevLogIndex,
//		PrevLogTerm:  prevLogTerm,
//		Entries:      rf.log[pos+1:],
//		LeaderCommit: rf.commitIndex,
//	}
//	rf.mu.Unlock()
//	reply := AppendEntriesReply{}
//	rf.sendAppendEntries(serverID, &args, &reply)
//}

func (rf *Raft) getPrev(serverID int) (int, int, int) {
	prevLogIndex := rf.leaderTool.nextIndex[serverID] - 1
	var prevLogTerm int
	var pos int
	for i := len(rf.log) - 1; i >= 0; i-- {
		if rf.log[i].Index == prevLogIndex {
			prevLogTerm = rf.log[i].Term
			pos = i
		}
	}
	return prevLogIndex, prevLogTerm, pos
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
	//rf.mu.Lock()
	//rf.lastHeartBeat = time.Now()//student guide说不需要
	//LeaderID := rf.me
	//考虑一种情况：如果当前Leader是旧leader,它向其他人发送heartbeat的时候，使用的参数必须是当前的term
	term := rf.currentTerm //需要把当前状态的term给拿到

	//rf.mu.Unlock()
	for serverID, _ := range rf.peers {
		if serverID != rf.me {
			go func(serverID int) {
				//nextIndex := rf.leaderTool.nextIndex[serverID]
				args := AppendEntriesArgs{
					Term:     term,
					LeaderID: rf.me,

					//设置prevLogIndex和Term,
					PrevLogIndex: rf.leaderTool.nextIndex[serverID] - 1,
					PrevLogTerm:  rf.log[rf.leaderTool.nextIndex[serverID]-1].Term,
					//log在外面设置
					//设置leaderCommit
					LeaderCommit: rf.commitIndex,
				}
				//考虑要不要同步日志，应该是不需要的
				reply := AppendEntriesReply{}
				//fmt.Printf("LeaderID:%d---向serverID:%d---发送心跳\n", LeaderID, serverID)
				ok := rf.sendAppendEntries(serverID, &args, &reply)
				rf.mu.Lock()
				defer rf.mu.Unlock()
				if ok && reply.Success == false {
					if reply.Term > rf.currentTerm {
						fmt.Printf("Term = %d,Leader = %d,收到来自node = %d的Term = %d,更新自己的Term\n", rf.currentTerm, rf.me, serverID, reply.Term)
						rf.setNewTermBybiggerTerm(reply.Term)
					} else {
						rf.leaderTool.nextIndex[serverID] = max(rf.leaderTool.nextIndex[serverID]-1, 1) //这里不用考虑执行顺序问题，只要属于majority，就不会出现不一致,但是要注意nextIndex最小为1
					}
				} else if reply.Success == true {
					rf.leaderTool.matchIndex[serverID] = max(args.PrevLogIndex, rf.leaderTool.matchIndex[serverID]) //说明上一条对的上
				}

			}(serverID)
		}
	}
}

func (rf *Raft) ticker() { //心跳机制
	count := 0
	for rf.killed() == false {
		//fmt.Printf("serverID:%d---进入ticker\n", rf.me)
		// Your code here (3A)
		// Check if a leader election should be started.

		// pause for a random amount of time between 50 and 350
		// milliseconds.

		time.Sleep(rf.heartBeat)
		//如果整体上锁，整个操作会具有原子性，但是我们不能在RPC的时候上锁等待回应
		//rf.mu.Lock()
		state := rf.state
		//rf.mu.Unlock()
		count++

		if state == Leader {
			if count >= 10 {
				count = 0
				fmt.Printf("Term= %d,Leader = %d,完成10次心跳\n", rf.currentTerm, rf.me)
			}
			go rf.sendHeartBeat() //应该用线程，否则会阻塞//heartbeat有等幂性，无所谓同时收到多条
			//
		} else {
			if count >= 10 {
				count = 0
				fmt.Printf("Term= %d,node = %d,完成10次心跳\n", rf.currentTerm, rf.me)
			}
			electionTimeout := GetElectionTimeout()
			//fmt.Printf("serverID:%d---状态为%s\n", rf.me, rf.state)
			//rf.mu.Lock()
			needElection := time.Since(rf.lastHeartBeat).Milliseconds() > electionTimeout
			//rf.mu.Unlock()
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
	//设置nextIndex以及matchIndex
	initNextIndex := rf.log[len(rf.log)-1].Index + 1
	for i, _ := range rf.leaderTool.nextIndex {
		rf.leaderTool.nextIndex[i] = initNextIndex
		rf.leaderTool.matchIndex[i] = 0
	}
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
		Term:         rf.currentTerm,
		CandidateID:  rf.me,
		LastLogIndex: rf.log[len(rf.log)-1].Index,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
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
				} else if reply.VoteGranted == false {
					fmt.Printf("term:%d,serverID:%d---收到了来自serverID:%d,对方拒绝了投票\n", args.Term, rf.me, serverID)
				} else {
					fmt.Printf("term:%d,serverID:%d---收到了来自serverID:%d,对方未接受到投票\n", args.Term, rf.me, serverID)
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

func (rf *Raft) applier() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	for rf.killed() == false {
		for rf.commitIndex > rf.lastApplied {
			rf.lastApplied++
			applyMsg := raftapi.ApplyMsg{
				CommandValid: true,
				Command:      rf.log[rf.lastApplied].Command,
				CommandIndex: rf.lastApplied,
			}
			rf.applyCh <- applyMsg
			//这里的发送需要考虑死锁风险,因为:我们发送给app，app需要上锁来应用，但是app刚才通过上锁来请求了raft执行这条之类，并且只有执行完成之后才能解锁
		}
		rf.applyCond.Wait()
	}
}
func (rf *Raft) apply() {
	rf.applyCond.Broadcast()
}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *tester.Persister, applyCh chan raftapi.ApplyMsg) raftapi.Raft {
	rf := &Raft{}
	//工具类的设置
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applyCh = applyCh
	rf.applyCond = sync.NewCond(&rf.mu)
	//rf.cond = sync.NewCond(&rf.mu)
	//持久化数据
	rf.currentTerm = 0
	rf.voteFor = -1
	rf.log = []LogEntry{
		{
			Term:    0,
			Index:   0,
			Command: struct{}{},
		},
	}

	//rf.log = Log{
	//	LogMap:    make(map[int]Entry),
	//	Length:    1,
	//	lastIndex: 0,
	//	lastTerm:  0,
	//}
	//rf.log.LogMap[0] = Entry{
	//	Term:    0,
	//	Command: struct{}{},
	//}
	//非持久化数据
	rf.commitIndex = 0
	rf.lastApplied = 0
	//Leader相关数据
	rf.leaderTool = LeaderTool{} //选上了再初始化
	//一些没有提到但是需要的
	rf.state = Follower
	rf.leaderID = -1
	rf.lastHeartBeat = time.Now()
	rf.leaderTool.nextIndex = make([]int, len(rf.peers))
	rf.leaderTool.matchIndex = make([]int, len(rf.peers))
	//心跳的时间间隔：
	rf.heartBeat = time.Duration(50) * time.Millisecond
	rf.minimumElectionTimeout = 100 * time.Millisecond
	//fmt.Printf("serverID:%d---创建\n", me)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()  //初始化完毕，投入工作
	go rf.applier() //不断检查commitIndex,提交apply信息

	return rf
}
