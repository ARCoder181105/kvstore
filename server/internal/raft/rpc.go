package raft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
