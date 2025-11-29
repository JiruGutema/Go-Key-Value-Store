# Reflection report 

**Concurrency model**

* The server uses a goroutine per client connection (in `for { conn, _ := listener.Accept(); go handleConnection(conn, store) }`).
* The key-value map is protected by a `sync.RWMutex` in `KVStore`.

  * `Put` and `Delete` use `mu.Lock()` to modify the map.
  * `Get` and `List` use `mu.RLock()` so multiple reads can proceed concurrently.
* `List()` returns a shallow copy so that callers don't hold references to the internal map, avoiding accidental races.

**Challenges faced & solutions**

1. **Concurrent updates to the map** — solved by using `sync.RWMutex`. This prevents race conditions if multiple clients update the same key.
2. **Parsing commands where values may contain spaces** — solved by extracting the value as the remainder of the input line after the key (not by simple `strings.Fields` alone).
3. **Client disconnects** — server handles `io.EOF`/read errors gracefully and closes the goroutine. The server keeps other goroutines running.
4. **Client reconnecting** — the client implements a simple reconnect loop when the connection fails; this keeps UX smooth for intermittent network issues.

**Data consistency**

* Using mutex locks ensures serializability for updates: `Put` and `Delete` operate under exclusive lock. `Get` and `List` operate under shared read locks. This avoids races and ensures clients see consistent values.
* For high throughput or for scaling to many nodes, one would consider sharding, optimistic concurrency, or an external consistent store. This implementation focuses on correctness and clarity for an in-memory single-node store.
