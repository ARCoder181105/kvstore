# 📘 KVStore Learning Guide & Implementation Checklist

So far, you've done an incredible job laying the foundation. We have a pure memory storage engine, a binary TCP protocol, a persistence layer, an HTTP API, and a beautiful Next.js frontend! 🚀

Right now, you are tackling **Phase 7: Raft Consensus**. Raft is one of the most rewarding algorithms to build, but also one of the easiest places to introduce subtle bugs.

Based on our analysis, here is the exact state of your project, what you left out, and how to approach finishing it—**without me writing the code for you**. 

---

## 🎯 Current Project State

- **Phase 1-6:** ✅ Largely Complete. The core store, TCP networking, and API wrappers are well-defined.
- **Phase 7 (Raft):** 🚧 In Progress. You are currently wrapping up **Week 10 / Week 11**.
  - `node.go` structures and basic states are initialized.
  - `log.go` basic helper methods are implemented.
  - `log_test.go` has unit tests for log operations.
- **Phase 8 (Observability):** ⏳ Not Started yet.

---

## ✅ The "Left Out Stuff" Checklist

Here is exactly what remains for you to build to finish this system:

### 🛠️ Raft Consensus (Weeks 10-13)
- [ ] **Election Timer Loop:** A background goroutine that constantly ticks and triggers an election if a heartbeat hasn't been received in 150-300ms.
- [ ] **RPC Structures:** Define `RequestVoteArgs`, `RequestVoteReply` and `AppendEntriesArgs`, `AppendEntriesReply`.
- [ ] **Candidate Logic (`startElection`):** Transitioning to candidate, incrementing term, voting for self, and sending `RequestVote` RPCs in parallel.
- [ ] **Follower Logic (`RequestVote` Handler):** Evaluating inbound votes. Granting a vote if the candidate's term is newer and their log is at least as up-to-date.
- [ ] **Leader Logic (`AppendEntries` Heartbeat):** Sending empty log entries every 50ms so followers don't trigger new elections.
- [ ] **Log Replication:** Leader accepts commands from clients, appends to its own log, and broadcasts them via `AppendEntries`.
- [ ] **Commit Index Management:** Advancing the `commitIndex` once a majority of nodes acknowledge a log entry.
- [ ] **State Machine Application:** A background loop (`applyCommitted()`) that reads from the `commitIndex` and physically executes those commands on the KV `Store`.
- [ ] **Leader Write Proxy:** Ensuring followers who receive arbitrary writes proxy them safely over to the active leader. 

### 📊 Observability (Week 14)
- [ ] **Prometheus Metrics:** Tracking command counts, latencies, etc.
- [ ] **Load Testing / Benchmarking (`bench/bench.go`):** Validating the >100K ops/sec throughput target.

---

## 🧭 Your Path Forward (Teacher's Guide)

Since my goal is to help you become a better programmer, I won’t write this code for you. Instead, let me act as your pair-programming instructor. 

### First Step: The Election Timer
You are currently inside `internal/raft/election.go` and `node.go`. Your very first job is to ensure a node doesn't stay a `Follower` forever.

**Stop and think about these questions:**
1. *In `node.go`, we have `electionTimeout` and `electionResetAt`. If a node boots up as a Follower, how does it know when to become a Candidate?*
2. *Raft demands "randomized election timeouts" (e.g., 150ms to 300ms). Why? What catastrophe would happen if all 3 nodes had an exact 150ms timeout?*

**Implementation Direction:**
Create a method `runElectionTimer()` that runs inside a goroutine as soon as `New(...)` is called. It should wake up every few milliseconds, check the current time against `node.electionResetAt` + `node.electionTimeout`. If elapsed, call a `startElection()` method! 

### Second Step: The Vote Request
Once your timer fires, your node is a candidate. It must ask for votes.

**Questions:**
1. *Before you ask for votes from peers, what two state properties must you change locally? (Hint: See section 5.2 of the Raft paper).*
2. *If a node receives a `RequestVote`, under what exact scenarios must it reject the vote AND under what scenarios must it accept?*

---

## 📚 Essential Reading

Do not write code for Raft without having the answers immediately available. Distribute systems rely purely on getting these edge cases right.

1. **[The Secret Lives of Data (Raft Visualization)](https://thesecretlivesofdata.com/raft/)**: Watch this animation completely. It's the best summary of everything you're about to write.
2. **[The Raft Paper](https://raft.github.io/raft.pdf)**: Read **Sections 5.1 through 5.4**. You do not need to read the whole paper, but these 4 pages are your "source of truth". They literally define the `if/else` statements you will write in Go.
3. Your local `docs/RAFT.md` file. It breaks down the architecture specifically mapped to *this* KV Store project's types!

---

Whenever you're ready, take a stab at creating the `runElectionTimer` mechanism or setting up the RPC structures. Let me know what you try, and I will be right here to review it and guide your logic!
