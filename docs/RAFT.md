# Raft Consensus

> Step-by-step guide to implementing the Raft distributed consensus algorithm on top of the KV store. Read the Raft paper first — this document is your implementation guide.

---

## Prerequisites

Before starting this phase:
- Phases 1–3 are completely done and `go test -race` passes
- You have read [the Raft paper](https://raft.github.io/raft.pdf) — at minimum sections 5.1 through 5.4
- You understand what "split-brain" means and why it is dangerous

---

## What Problem Raft Solves

You have 3 server nodes. All three store the same data. A client sends `SET counter 42` to Node A.

Questions:
1. What if Node A writes it but crashes before telling Nodes B and C?
2. What if the network between A and B drops — both think they are the leader?
3. What if Node C is 5 seconds behind and serves a stale read?

Raft answers all of these. Its guarantee: **if a command is confirmed to the client, a majority of nodes have it in their log, even if some nodes crash**.

---

## The Three Core Mechanisms

### 1. Leader Election

Only one node can be leader at a time. The leader is the only node that accepts writes.

A node starts as a **Follower**. If it does not hear from a leader for a random timeout (150–300ms), it becomes a **Candidate**, increments its term, and sends `RequestVote` RPCs to all peers. If it gets votes from a majority (2 out of 3 nodes), it becomes the **Leader** and starts sending heartbeats.

```
           No heartbeat for election timeout
FOLLOWER ─────────────────────────────────► CANDIDATE
    ▲                                            │
    │ Discovers valid leader                     │ Receives majority votes
    │ or higher term number                      ▼
    └────────────────────────────────────── LEADER
                                                 │
                                     Sends heartbeats every 50ms
                                     Replicates log entries
```

### 2. Log Replication

When a client sends a write command to the leader:
1. Leader appends the command to its log (not yet committed)
2. Leader sends `AppendEntries` RPCs to all followers with the new log entry
3. When a majority acknowledge (2 out of 3 including the leader), the leader marks it as **committed**
4. Leader applies the committed entry to the local KV store
5. Leader tells followers the commit index in the next heartbeat
6. Followers apply committed entries to their local KV stores
7. Leader responds OK to the client

### 3. Safety

A leader can only be elected if its log is at least as up-to-date as a majority of nodes. This guarantees that no committed entry is ever lost — the new leader will always have all committed entries.

---

## State Machine Per Node

```go
type RaftNode struct {
    // Persistent state (must survive crashes — write to disk before responding to RPCs)
    currentTerm uint64   // latest term seen
    votedFor    string   // candidateId voted for in current term ("" if none)
    log         []LogEntry

    // Volatile state
    commitIndex uint64   // highest log entry known to be committed
    lastApplied uint64   // highest log entry applied to state machine

    // Volatile leader state (reinitialize after election)
    nextIndex  map[string]uint64  // for each follower: next log index to send
    matchIndex map[string]uint64  // for each follower: highest log index confirmed replicated

    state NodeState
    id    string
    peers []string
    store *store.Store
}
```

---

## Week-by-Week Implementation

### Week 10 — Foundation

**Build:**

1. Create `internal/raft/node.go`
   - Define all structs above
   - Implement `NewNode(id string, peers []string, store *store.Store) *RaftNode`
   - Implement state transitions: `becomeFollower(term)`, `becomeCandidate()`, `becomeLeader()`
   - Implement persistent state save/load (write `currentTerm` and `votedFor` to disk on every change)

2. Create `internal/raft/log.go`
   - `appendEntry(entry LogEntry)` — add to log
   - `getEntry(index uint64) LogEntry`
   - `getLastIndex() uint64` and `getLastTerm() uint64`
   - `truncateFrom(index uint64)` — remove conflicting entries (used when a follower receives entries that conflict with its log)

3. Create election timer logic in `node.go`
   - Random timeout between 150–300ms (randomness prevents all nodes becoming candidates at the same time)
   - Reset timer on every valid heartbeat received
   - On timeout: call `becomeCandidate()`

**End-of-week check:** Unit test that a node starts as Follower, and transitions to Candidate after the election timeout passes without a heartbeat.

---

### Week 11 — Leader Election

**Build:** `internal/raft/rpc.go` (RequestVote)

1. Define RPC structs:
```go
type RequestVoteArgs struct {
    Term         uint64
    CandidateID  string
    LastLogIndex uint64
    LastLogTerm  uint64
}

type RequestVoteReply struct {
    Term        uint64
    VoteGranted bool
}
```

2. Implement `RequestVote` handler (this runs on the node receiving the vote request):
```
If args.Term < node.currentTerm: reject
If already voted for someone else in this term: reject
If candidate's log is less up-to-date than ours: reject
Otherwise: grant vote, update votedFor, reset election timer
```

3. Implement the candidate's side: `startElection()`
   - Increment `currentTerm`
   - Vote for self
   - Send `RequestVote` to all peers in parallel (goroutines)
   - Count votes as replies come in
   - If majority votes received: `becomeLeader()`
   - If any reply has higher term: `becomeFollower(reply.Term)`

4. Implement HTTP handlers for RPCs (POST `/raft/requestvote`)

**End-of-week check:** Start 3 nodes in a test. One should be elected leader within 500ms. Kill the leader — a new one elected within 500ms.

---

### Week 12 — Log Replication

**Build:** `internal/raft/rpc.go` (AppendEntries)

1. Define RPC structs:
```go
type AppendEntriesArgs struct {
    Term         uint64
    LeaderID     string
    PrevLogIndex uint64
    PrevLogTerm  uint64
    Entries      []LogEntry
    LeaderCommit uint64
}

type AppendEntriesReply struct {
    Term    uint64
    Success bool
}
```

2. Implement `AppendEntries` handler (runs on followers):
```
If args.Term < node.currentTerm: reject
If log doesn't contain entry at PrevLogIndex with PrevLogTerm: reject
If conflicting entries exist: truncate log from conflict point
Append new entries
If leaderCommit > commitIndex: update commitIndex
Reset election timer (this is a valid heartbeat)
```

3. Implement leader's heartbeat loop:
   - Every 50ms, send `AppendEntries` to all followers (with empty entries if no new commands)
   - Include `LeaderCommit` so followers can advance their commit index

4. Implement leader's log replication on new command:
   - Append entry to own log
   - Send `AppendEntries` to all followers
   - When majority acknowledge: increment `commitIndex`
   - Apply committed entries to store: call `store.Set(...)` for each newly committed entry

**End-of-week check:** Leader commits a SET command. Kill the leader. The new leader has the committed entry.

---

### Week 13 — Commit, Apply, and Leader Proxy

**Build:** Complete end-to-end write path

1. Implement `applyCommitted()` — runs in a goroutine, watches `commitIndex`, applies entries up to `lastApplied`:
```go
for lastApplied < commitIndex {
    lastApplied++
    entry := log[lastApplied]
    cmd := protocol.DecodeCommand(entry.Command)
    executeCommand(cmd, store)
}
```

2. Implement **leader proxy** for followers:
   - If a follower receives a write request via HTTP API, it looks up the current leader and proxies the request
   - Response comes back to the original client transparently

3. Implement client read semantics:
   - For now: reads can go to any node (potentially slightly stale)
   - Stretch goal: linearizable reads require the leader to confirm it is still the leader before serving the read

4. Wire Raft into `cmd/server/main.go` with a `--raft-enabled` flag

**End-of-week test:**
```bash
./kvstore-server --config node1.yaml &  # becomes leader
./kvstore-server --config node2.yaml &
./kvstore-server --config node3.yaml &

./kvcli --port 6379 SET city Mumbai   # → OK
kill <node1-pid>                       # kill leader
sleep 3                                # wait for election
./kvcli --port 6380 GET city          # → Mumbai (committed, replicated)
```

---

## Common Raft Bugs

**Bug 1: Not persisting `votedFor` before responding**
If a node votes for Candidate A and crashes before saving `votedFor`, it might vote for Candidate B in the same term after restart. Solution: always write `currentTerm` and `votedFor` to disk before sending any RPC response.

**Bug 2: Off-by-one in log indexing**
Raft logs are 1-indexed. Index 0 is a sentinel "empty" entry with term 0. Getting the PrevLogIndex/PrevLogTerm comparison wrong by one causes followers to reject valid entries. Write a test specifically for the boundary case where a follower has an empty log.

**Bug 3: Not resetting the election timer on valid AppendEntries**
If a follower does not reset its timer when it receives a valid heartbeat, it will start a spurious election. The timer must reset even for heartbeats with empty entries.

**Bug 4: Applying entries before they are committed**
Only apply entries to the store when `lastApplied < commitIndex`. Applying before the commit index is confirmed means you might apply an entry that gets overwritten.

**Bug 5: Split vote loops**
If all nodes start at the same time, they all timeout at the same time, all become candidates, all vote for themselves, and no one gets a majority. The random election timeout (150–300ms) prevents this. Make sure your random source is seeded differently per node.

---

## Testing Raft

Unit tests alone are not enough for Raft. You need chaos tests:

```go
// Test: partition the leader from the rest of the cluster
func TestLeaderPartition(t *testing.T) {
    cluster := startCluster(3)
    leader := cluster.waitForLeader()
    
    // Partition: block all network between leader and followers
    cluster.partition(leader, cluster.others(leader))
    
    // Followers should elect a new leader
    newLeader := cluster.waitForLeaderExcluding(leader, 3*time.Second)
    assert.NotEqual(t, leader, newLeader)
    
    // Old leader should step down when partition heals
    cluster.healPartition()
    time.Sleep(500 * time.Millisecond)
    assert.Equal(t, Follower, leader.state)
}
```

---

## Resources for Raft

- [The Raft Paper](https://raft.github.io/raft.pdf) — the original paper, sections 5.1–5.4 are the implementation guide
- [The Raft Visualization](https://raft.github.io) — interactive animation, essential for understanding leader election
- [MIT 6.5840 Lab 2](https://pdos.csail.mit.edu/6.5840/labs/lab-raft.html) — the famous Raft lab from MIT's distributed systems course (do not copy the code, but the test cases are a great reference for what you need to handle)
- [etcd's raft library](https://github.com/etcd-io/raft) — production implementation, good reference for edge cases
