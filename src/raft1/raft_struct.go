package raft

const (
	Follower  = "follower"
	Candidate = "candidate"
	Leader    = "leader"
)

type LogEntry struct {
	Term    int
	Index   int
	Command interface{}
}

//	type Entry struct {
//		Term    int
//		Command interface{}
//	}
//
//	type Log struct {
//		LogMap    map[int]Entry
//		Length    int
//		lastIndex int
//		lastTerm  int
//	}
type LeaderTool struct {
	nextIndex  []int
	matchIndex []int
}
type AppendEntriesArgs struct {
	//基本的
	Term     int
	LeaderID int
	//日志复制需要的
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	//状态机提交需要的
	LeaderCommit int
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term        int
	CandidateID int
	//选举限制
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int
	VoteGranted bool
	//根据term就可以识别失败的类型，任期相同，那么只能有自己一个Leader，那么肯定是日志对不上
}

//func (log *Log) append(command interface{}, term int) int {
//	log.LogMap[log.lastIndex+1] = Entry{
//		Term:    term,
//		Command: command,
//	}
//	log.lastIndex++
//	log.lastTerm = term
//	log.Length++
//	return log.lastIndex
//}
