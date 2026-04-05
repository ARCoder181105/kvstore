package raft

import "time"

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

	r.mu.Unlock()
}
