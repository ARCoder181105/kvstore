package raft

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
