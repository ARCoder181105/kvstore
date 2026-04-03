package raft

import (
	"sync"
	"time"
)

type NodeID string

const NoVote NodeID = ""

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

type LogEntry struct {
	Term    uint64
	Command interface{}
}

type RaftNode struct {
	mu sync.Mutex

	// identity & cluster
	id    NodeID
	peers []NodeID

	// role
	state State

	// persistent state
	currentTerm uint64
	votedFor    NodeID // NoVote means not voted in this term
	log         []LogEntry

	// volatile state
	commitIndex uint64
	lastApplied uint64

	// leader state
	nextIndex  map[NodeID]uint64
	matchIndex map[NodeID]uint64

	// leader tracking
	leaderID NodeID

	// timers
	electionTimeout  time.Duration
	heartbeatTimeout time.Duration
	electionResetAt  time.Time

	// apply committed entries to state machine
	applyCh chan LogEntry

	// pending client requests (index -> response channel)
	pending map[uint64]chan interface{}
}
