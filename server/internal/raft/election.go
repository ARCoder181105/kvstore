package raft

import (
	"fmt"
	"time"
)

func (r *RaftNode) runElectionTimer() {

	for {
		time.Sleep(10 * time.Millisecond)

		r.mu.Lock()
		shouldStartElection := time.Since(r.electionResetAt) >= r.electionTimeout && r.state != Leader
		r.mu.Unlock()

		if shouldStartElection {
			r.startElection()
		}
	}

}

func (r *RaftNode) startElection() {
	r.mu.Lock()

	r.currentTerm += 1
	r.state = Candidate
	r.votedFor = r.id

	term := r.currentTerm
	lastIndex := r.getLastIndex()
	lastTerm := r.getLastTerm()

	r.mu.Unlock()

	votes := 1

	for k, v := range r.peers {
		go func(peerID NodeID, peerURL string) {
			reply, err := r.sendRequestVote(peerURL, RequestVoteArgs{
				Term:         term,
				CandidateID:  r.id,
				LastLogIndex: lastIndex,
				LastLogTerm:  lastTerm,
			})

			if err != nil {
				fmt.Printf("requestvote to %v failed: %v\n", peerID, err)
				return
			}

			r.mu.Lock()
			defer r.mu.Unlock()

			if reply.Term > r.currentTerm {
				r.currentTerm = reply.Term
				r.state = Follower
				r.votedFor = NoVote
				r.leaderID = ""
				r.electionResetAt = time.Now()
				return
			}

			if r.state != Candidate || r.currentTerm != term {
				return
			}

			if reply.VoteGranted {
				votes += 1

				if votes > (len(r.peers)+1)/2 {
					r.state = Leader
					r.electionResetAt = time.Now()
				}
			}
		}(k, v)
	}

}
