package raft

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ARCoder181105/kvstore/internal/metrics"
	aof "github.com/ARCoder181105/kvstore/internal/persistence"
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

	store       *store.Store
	aofWriter   *aof.AOFWriter
	commitReady chan struct{}
}

func New(id NodeID, peers map[NodeID]string, store *store.Store, aofWriter *aof.AOFWriter) *RaftNode {
	r := &RaftNode{
		id:              id,
		state:           Follower,
		votedFor:        NoVote,
		log:             []LogEntry{{Index: 0, Term: 0}}, // sentinel entry
		pending:         make(map[uint64]chan interface{}),
		applyCh:         make(chan LogEntry, 1024),
		nextIndex:       make(map[NodeID]uint64),
		matchIndex:      make(map[NodeID]uint64),
		peers:           peers,
		store:           store,
		aofWriter:       aofWriter,
		electionResetAt: time.Now(),
		electionTimeout: time.Duration(150+rand.Intn(150)) * time.Millisecond,
		commitReady:     make(chan struct{}, 1),
	}

	go r.runElectionTimer()
	go r.applyCommitted()

	return r
}

type ErrorNotLeader struct {
	LeaderID NodeID
}

func (e *ErrorNotLeader) Error() string {
	return fmt.Sprintf("not leader, leader is %s", e.LeaderID)
}

func (r *RaftNode) Submit(cmd protocol.Command) (interface{}, error) {
	r.mu.Lock()
	if r.state != Leader {
		leader := r.leaderID
		r.mu.Unlock()
		return nil, &ErrorNotLeader{LeaderID: leader}
	}

	index := uint64(len(r.log))
	term := r.currentTerm

	waitCh := make(chan interface{}, 1)
	r.pendingMu.Lock()
	r.pending[index] = waitCh
	r.pendingMu.Unlock()

	r.log = append(r.log, LogEntry{
		Index:   index,
		Term:    term,
		Command: cmd,
	})
	r.mu.Unlock()

	select {
	case res := <-waitCh:
		return res, nil
	case <-time.After(2 * time.Second):
		r.pendingMu.Lock()
		delete(r.pending, index)
		r.pendingMu.Unlock()
		return nil, fmt.Errorf("raft submit timeout")
	}
}

func (r *RaftNode) applyCommitted() {
	for {
		r.mu.Lock()

		// If there is nothing to apply, unlock and wait for a signal
		for r.commitIndex <= r.lastApplied {
			r.mu.Unlock()
			<-r.commitReady // Go scheduler puts this goroutine to sleep here!
			r.mu.Lock()
		}

		var commandsToApply []LogEntry
		for r.commitIndex > r.lastApplied {
			r.lastApplied++
			entry := r.log[r.lastApplied]
			commandsToApply = append(commandsToApply, entry)
		}
		r.mu.Unlock()

		for _, entry := range commandsToApply {
			cmd := entry.Command
			switch cmd.ID {
			case protocol.CmdSet:
				r.store.Set(cmd.Key, cmd.Value, cmd.TTL)
				metrics.CommandsTotal.WithLabelValues(raftCommandName(cmd.ID)).Inc()
				if r.aofWriter != nil {
					var expiresAt int64
					if cmd.TTL > 0 {
						expiresAt = time.Now().UnixNano() + cmd.TTL
					}
					r.aofWriter.Append(aof.AOFEntry{
						Timestamp: time.Now().UnixNano(),
						CmdID:     protocol.CmdSet,
						Key:       cmd.Key,
						Value:     cmd.Value,
						ExpiresAt: expiresAt,
					})
				}

			case protocol.CmdDel:
				r.store.Delete(cmd.Key)
				metrics.CommandsTotal.WithLabelValues(raftCommandName(cmd.ID)).Inc()
				if r.aofWriter != nil {
					r.aofWriter.Append(aof.AOFEntry{
						Timestamp: time.Now().UnixNano(),
						CmdID:     protocol.CmdDel,
						Key:       cmd.Key,
					})
				}

			case protocol.CmdExpire:
				r.store.Expire(cmd.Key, cmd.TTL)
				metrics.CommandsTotal.WithLabelValues(raftCommandName(cmd.ID)).Inc()
				if r.aofWriter != nil {
					var expiresAt int64
					if cmd.TTL > 0 {
						expiresAt = time.Now().UnixNano() + cmd.TTL
					}
					r.aofWriter.Append(aof.AOFEntry{
						Timestamp: time.Now().UnixNano(),
						CmdID:     protocol.CmdExpire,
						Key:       cmd.Key,
						ExpiresAt: expiresAt,
					})
				}
			}

			r.pendingMu.Lock()
			ch, ok := r.pending[entry.Index]
			if ok {
				ch <- struct{}{}
				delete(r.pending, entry.Index)
			}
			r.pendingMu.Unlock()
		}

		// Update key count gauge once after the whole batch
		if r.store != nil {
			metrics.KeysTotal.Set(float64(r.store.Count()))
		}
	}
}

func (r *RaftNode) GetPeerURL(id NodeID) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	url, ok := r.peers[id]
	return url, ok
}

func raftCommandName(id byte) string {
	switch id {
	case protocol.CmdSet:
		return "set"
	case protocol.CmdDel:
		return "del"
	case protocol.CmdExpire:
		return "expire"
	default:
		return "unknown"
	}
}
