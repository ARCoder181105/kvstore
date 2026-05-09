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
	r.electionResetAt = time.Now()

	term := r.currentTerm
	lastIndex := r.getLastIndex()
	lastTerm := r.getLastTerm()

	r.mu.Unlock()

	votes := 1

	if len(r.peers) == 0 {
		r.mu.Lock()
		r.state = Leader
		r.electionResetAt = time.Now()
		r.leaderID = r.id
		// nextIndex/matchIndex are empty
		r.mu.Unlock()
		go r.runHeartbeatLoop()
		return
	}

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
					r.leaderID = r.id

					lastIdx := uint64(len(r.log))
					for pID := range r.peers {
						r.nextIndex[pID] = lastIdx
						r.matchIndex[pID] = 0
					}

					go r.runHeartbeatLoop()
				}
			}
		}(k, v)
	}

}

func (r *RaftNode) advanceCommitIndex() {
	for n := uint64(len(r.log)) - 1; n > r.commitIndex; n-- {
		if r.log[n].Term == r.currentTerm {
			matches := 1 // self
			for pID := range r.peers {
				if r.matchIndex[pID] >= n {
					matches++
				}
			}
			if matches > (len(r.peers)+1)/2 {
				r.commitIndex = n

				select {
				case r.commitReady <- struct{}{}:
				default:
				}

				return
			}
		}
	}
}

func (r *RaftNode) runHeartbeatLoop() {
	for {
		time.Sleep(50 * time.Millisecond)

		r.mu.Lock()
		if r.state != Leader {
			r.mu.Unlock()
			return
		}

		term := r.currentTerm
		id := r.id
		leaderCommit := r.commitIndex

		for peerID, peerURL := range r.peers {
			nextIdx := r.nextIndex[peerID]
			prevLogIndex := nextIdx - 1

			// getEntry handles out of bounds safely
			var prevLogTerm uint64
			if prevLogIndex < uint64(len(r.log)) {
				prevLogTerm = r.log[prevLogIndex].Term
			}

			entries := make([]LogEntry, 0)
			if nextIdx < uint64(len(r.log)) {
				entries = append(entries, r.log[nextIdx:]...)
			}

			go func(pID NodeID, pURL string, nIdx uint64, pIdx uint64, pTerm uint64, e []LogEntry) {
				reply, err := r.sendAppendEntries(pURL, AppendEntriesArgs{
					Term:         term,
					LeaderID:     id,
					PrevLogIndex: pIdx,
					PrevLogTerm:  pTerm,
					Entries:      e,
					LeaderCommit: leaderCommit,
				})

				if err != nil {
					return
				}

				r.mu.Lock()
				defer r.mu.Unlock()

				if r.state != Leader || r.currentTerm != term {
					return
				}

				if reply.Term > r.currentTerm {
					r.currentTerm = reply.Term
					r.state = Follower
					r.votedFor = NoVote
					r.leaderID = ""
					return
				}

				if reply.Success {
					if len(e) > 0 {
						newMatch := pIdx + uint64(len(e))
						if newMatch > r.matchIndex[pID] {
							r.matchIndex[pID] = newMatch
						}
						r.nextIndex[pID] = r.matchIndex[pID] + 1
						r.advanceCommitIndex()
					}
				} else {
					if r.nextIndex[pID] > 1 {
						r.nextIndex[pID]--
					}
				}
			}(peerID, peerURL, nextIdx, prevLogIndex, prevLogTerm, entries)
		}

		// Advance commit index for the leader itself.
		// This is critical for single-node deployments where the peer loop
		// above never executes, so commitIndex would never advance otherwise.
		r.advanceCommitIndex()
		r.mu.Unlock()
	}
}
