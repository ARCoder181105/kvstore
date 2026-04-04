package raft

import (
	"sync"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/ARCoder181105/kvstore/internal/store"
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
	Index   uint64
	Term    uint64
	Command protocol.Command
}

type RaftNode struct {
	mu        sync.Mutex
	pendingMu sync.Mutex

	// identity & cluster
	id    NodeID
	peers map[NodeID]string // NodeID → "http://localhost:7001"

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

	store *store.Store
}
