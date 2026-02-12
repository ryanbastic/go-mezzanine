 How do I complete a live shard migration and rebalancing with this architecture?

● Let me look at the current shard routing and storage layer to give you a grounded answer.

● Explore(Explore shard routing architecture)
  ⎿  Done (23 tool uses · 53.0k tokens · 55s)

● Good question. There's no built-in rebalancing in go-mezzanine today. Here's how the current architecture constrains you:

  What you have now:
  - Shard assignment is FNV32(row_key) % numShards — deterministic and fixed
  - shards.json statically maps contiguous shard ranges to backends
  - shard.Router is a simple map[ID]CellStore protected by sync.RWMutex
  - Tables are named cells_NNNN, one per shard per backend

  Why live rebalancing is hard with this design:
  - Changing numShards changes ForRowKey() output for every row — you'd need to rehash and move all data
  - added_id is a per-table BIGSERIAL, so it doesn't transfer meaningfully across backends

  ---
  What you can do: Move whole shard ranges between backends

  Since each shard is a self-contained table (cells_0512), you can relocate entire shard ranges without changing numShards. This is the practical path:

  1. Prepare the target backend

  Create the shard tables on the new backend by running RunMigrationsForPool for the shard range you're moving (e.g., shards 2048–4095 from db2 → db3).

  2. Copy data (while writes continue to old backend)

  Use pg_dump/pg_restore per table, or logical replication, to copy cells_2048 through cells_4095 to the new backend. You can also use the existing partitionRead endpoint
  to stream cells by added_id.

  3. Catch up the delta

  After the bulk copy, do a second pass copying rows where added_id > last_copied_id (or created_at > cutoff). Repeat until the gap is small.

  4. Cut over (brief write pause)

  - Stop writes to the affected shards (or queue them)
  - Do a final delta copy
  - Update shards.json to point the shard range at the new backend
  - Restart the server (or, if you add hot-reload to Router, swap the mapping live)
  - Resume writes

  5. Drop old tables

  Once traffic is confirmed on the new backend, drop the migrated tables from the old one.

  ---
  If you want zero-downtime live migration

  You'd need to add a few things to the codebase:

  1. Dual-write mode in Router — For migrating shards, write to both old and new backends. Reads continue from old until cutover.
  2. Hot config reload — Watch shards.json or expose an admin endpoint to call Router.Register() for remapped shards without restarting.
  3. A migration worker — A background process that uses partitionRead (scan by added_id) to copy cells from source to destination shard tables, tracking a high-water mark.
  4. Shard state machine — Track per-shard state: serving → migrating (dual-write) → draining → serving on new backend.

  ---
  What you should NOT do

  - Don't change numShards on a live system. The hash function FNV32(key) % N means every key would rehash to a different shard. This is a full data reshuffle.
  - Don't try consistent hashing as a bolt-on. It would require changing ForRowKey, ForKey, and every caller. If you want consistent hashing, it's a design-time decision
  for a future version.

  The simplest practical approach is: over-provision shards initially (e.g., 4096), keep them lightweight, and move whole ranges between backends as you scale. This is
  essentially how Uber's Schemaless (which inspired this) handles it.
