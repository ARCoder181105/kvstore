package raft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RequestVoteArgs struct {
	Term         uint64
	CandidateID  NodeID
	LastLogIndex uint64
	LastLogTerm  uint64
}

type RequestVoteReply struct {
	Term        uint64
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         uint64
	LeaderID     NodeID
	PrevLogIndex uint64
	PrevLogTerm  uint64
	Entries      []LogEntry
	LeaderCommit uint64
}

type AppendEntriesReply struct {
	Term    uint64
	Success bool
}

func (r *RaftNode) RequestVote(args RequestVoteArgs) RequestVoteReply {
	r.mu.Lock()
	defer r.mu.Unlock()

	if args.Term > r.currentTerm {
		r.currentTerm = args.Term
		r.votedFor = NoVote
		r.state = Follower
	}

	if args.Term < r.currentTerm {
		return RequestVoteReply{
			Term:        r.currentTerm,
			VoteGranted: false,
		}
	}

	if r.votedFor != NoVote && r.votedFor != args.CandidateID {
		return RequestVoteReply{
			Term:        r.currentTerm,
			VoteGranted: false,
		}
	}

	lastTerm := r.getLastTerm()
	lastIndex := r.getLastIndex()
	if lastTerm > args.LastLogTerm || (lastTerm == args.LastLogTerm && lastIndex > args.LastLogIndex) {
		return RequestVoteReply{
			Term:        r.currentTerm,
			VoteGranted: false,
		}
	}

	r.votedFor = args.CandidateID
	return RequestVoteReply{
		Term:        r.currentTerm,
		VoteGranted: true,
	}
}

func (r *RaftNode) sendRequestVote(peerURL string, args RequestVoteArgs) (RequestVoteReply, error) {
	var reply RequestVoteReply

	payload, err := json.Marshal(args)
	if err != nil {
		return reply, err
	}

	endpoint := strings.TrimRight(peerURL, "/") + "/raft/requestvote"
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		return reply, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return reply, fmt.Errorf("request vote failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return reply, err
	}

	return reply, nil
}

func (r *RaftNode) AppendEntries(args AppendEntriesArgs) AppendEntriesReply {
	r.mu.Lock()
	defer r.mu.Unlock()

	if args.Term < r.currentTerm {
		return AppendEntriesReply{
			Term:    r.currentTerm,
			Success: false,
		}
	}

	if args.Term > r.currentTerm {
		r.currentTerm = args.Term
		r.votedFor = NoVote
		r.state = Follower
	}

	r.state = Follower
	r.leaderID = args.LeaderID
	r.electionResetAt = time.Now()

	// Check PrevLogIndex / PrevLogTerm match.
	entry := r.getEntry(args.PrevLogIndex)
	if entry.Term != args.PrevLogTerm {
		return AppendEntriesReply{
			Term:    r.currentTerm,
			Success: false,
		}
	}

	// Replace conflicting suffix and append new entries.
	r.truncateFrom(args.PrevLogIndex + 1)
	r.log = append(r.log, args.Entries...)

	if args.LeaderCommit > r.commitIndex {
		lastIndex := r.getLastIndex()
		if args.LeaderCommit < lastIndex {
			r.commitIndex = args.LeaderCommit
		} else {
			r.commitIndex = lastIndex
		}

		select {
		case r.commitReady <- struct{}{}:
		default:
		}
	}

	return AppendEntriesReply{
		Term:    r.currentTerm,
		Success: true,
	}
}

func (r *RaftNode) sendAppendEntries(peerURL string, args AppendEntriesArgs) (AppendEntriesReply, error) {
	var reply AppendEntriesReply

	payload, err := json.Marshal(args)
	if err != nil {
		return reply, err
	}

	endpoint := strings.TrimRight(peerURL, "/") + "/raft/appendentries"
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		return reply, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return reply, fmt.Errorf("append entries failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return reply, err
	}

	return reply, nil
}
