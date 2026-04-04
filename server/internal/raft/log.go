package raft

func (r *RaftNode) getLastIndex() uint64 {
	if len(r.log) == 0 {
		return 0
	}
	
	return uint64(len(r.log)) - 1
}

func (r *RaftNode) getLastTerm() uint64 {
	if len(r.log) == 0 {
		return 0
	}

	return r.log[len(r.log)-1].Term
}

func (r *RaftNode) getEntry(index uint64) LogEntry {

	if index < uint64(len(r.log)) {
		return r.log[index]
	}

	return LogEntry{}
}

func (r *RaftNode) appendEntry(entry LogEntry) {

	r.log = append(r.log, entry)
}

func (r *RaftNode) truncateFrom(index uint64) {
	if index >= uint64(len(r.log)) {
		return
	}
	r.log = r.log[:index]
}
