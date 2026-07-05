# ⚡ Google SWE Intern 2027 Quick Reference Cheat Sheet (KVStore)

> **Quick review before your 45-min interview tomorrow!**

---

## 🎯 1. The 90-Second Intro (Say This!)
*"I built a production-grade distributed key-value store from scratch in Go, implementing core concepts from Redis and etcd without external frameworks.*

*At the storage layer, I built a **16-shard concurrent hash map** routed via **FNV-1a hashing** with per-shard `sync.RWMutex`, eliminating global lock contention. For key expiration, I designed an **O(log N) min-heap background worker** paired with lazy O(1) checks on read.*

*For high availability, I implemented the **Raft consensus algorithm** across a 3-node cluster, handling randomized leader election (150-300ms) and log replication with majority quorum (2/3). I also built a custom **binary TCP protocol** on port 6379, an HTTP/WebSocket REST API with live channel fan-out, and a **Next.js 15 real-time dashboard** with Prometheus/Grafana monitoring.*

*In parallel benchmark testing across 16 goroutines, the engine achieved **~62M combined ops/sec** (~11M Sets/sec + ~51M Gets/sec), scaling linearly with CPU cores."*

---

## 📊 2. Why This Project is a Cheat Code for SWE Interns
- **Proves CS Fundamentals:** Shows you master Big-O analysis ($O(\log N)$ min-heap vs $O(N)$ sweeps), OS synchronization (`RWMutex` vs `Mutex`), and networking (TCP byte streams).
- **Top 0.01% Intern Project:** While most candidates show CRUD web apps, you are showing a distributed consensus database that hits 62M ops/sec!

---

## 🧠 3. Key CS Fundamentals & Trade-Offs
| Question | Your Intern-Level Answer |
| :--- | :--- |
| **Why build from scratch?** | To master core CS fundamentals: thread synchronization, Big-O memory eviction algorithms, and TCP framing byte streams. |
| **Why `RWMutex` over `Mutex`?** | In databases, reads outnumber writes. Standard `Mutex` blocks everyone. `RWMutex` allows **unlimited concurrent readers** (`RLock`), only locking exclusively for writes. |
| **Why Min-Heap over O(N) sweeps?** | $O(N)$ sweeps freeze databases. A min-heap priority queue gives $O(\log N)$ timer insertion and $O(1)$ root peeking! |
| **Why Raft over 2PC?** | 2PC blocks indefinitely if the coordinator crashes. Raft guarantees consensus as long as a majority ($2/3$) of nodes are alive. |

---

## 🛠️ 4. What Did You Test?
1. **Concurrent Race Detection:** Ran all tests with `go test -race` to prove zero data races across goroutines.
2. **TTL Overwrite Edge Case:** Tested setting Key A with TTL, immediately overwriting without TTL, ensuring the min-heap timer doesn't evict the new persistent key (`TestSetOverwriteClearsTTL`).
3. **Slow Subscriber Edge Case:** Verified that a browser WebSocket subscriber that stops reading never blocks core `Set()`/`Delete()` mutexes (`TestSlowSubscriberDoesNotBlockStore`).
4. **Raft Network Partitions:** Simulated isolating the leader, verifying followers elect a new leader in ~300ms, and verifying the old leader steps down and truncates uncommitted logs when healed (`TestLeaderPartition`).

---

## 💻 5. Transitioning to the DSA Coding Problem
In an intern R1, after 10 mins of discussing your project, you'll get a 35-min LeetCode Medium/Hard DSA coding problem:
- **State Big-O Out Loud:** Before coding, explicitly state time and space complexity just like you did for your Min-Heap TTL.
- **Ask About Edge Cases Early:** Ask about empty arrays or duplicates before writing code (just like you tested 1-indexed Raft log boundaries!).
- **Think Out Loud:** Never stay silent! Keep a constant dialogue with your interviewer.
