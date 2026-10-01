# Stonelinks/honker: grond's fork of the honker Go binding

This is a fork of [russellromney/honker](https://github.com/russellromney/honker)
kept for one consumer, grond, which uses
the Go binding in `packages/honker-go`. The rest of the monorepo is untouched
upstream code. Upstream's own README is [`README.md`](../README.md) at the
repository root.

Each `grond-<upstream sha, 8 chars>` branch is upstream `main` at that commit plus the commits listed below (the plain `grond` branch is the older 217d1f2 base, kept so its pseudo-versions keep resolving), one commit
per patch, so it rebases onto a new upstream with
`git fetch upstream && git rebase upstream/main`. The `main` branch, when
present, mirrors upstream and carries no patches.

## Using it

The Go module keeps upstream's path, `github.com/russellromney/honker-go`, so
no import changes. Point a `replace` at this repository's subdirectory module
and pin a commit on the newest `grond-<sha>` branch:

```
replace github.com/russellromney/honker-go => github.com/Stonelinks/honker/packages/honker-go v0.0.0-<UTC commit time>-<12-char sha>
```

`go list -m -json github.com/Stonelinks/honker/packages/honker-go@<sha>` prints
the exact pseudo-version for a commit.

The Go binding needs a honker extension built from the same upstream commit the
branch is based on: the binding calls extension symbols by name
(`honker_watcher_open_v2`, `honker_lock_renew`, the 7-argument
`honker_scheduler_register`, ...), so a binding and an extension from different
commits can disagree on that ABI.

## The patches

Each one is something another honker binding already does and the Go binding
never got. None is a grond invention. They are not upstream because grond's
maintainer chose to carry a fork rather than send patches upstream; if upstream
lands an equivalent, the matching commit is dropped in the next rebase.

1. **Begin every transaction IMMEDIATE.** `Open` passes `_txlock=immediate` in
   the DSN so `database/sql` issues `BEGIN IMMEDIATE`. A deferred transaction
   that reads first and writes later fails a lock upgrade with `SQLITE_BUSY`
   immediately (busy_timeout does not cover upgrades), which shows up as
   "database is locked" when two processes write at once. honker-core, the
   Rust crate and the node/bun bindings already begin immediate.

2. **`Stream.SubscribeFrom` and no offset save without a consumer.**
   `SubscribeFrom(ctx, startOffset)` delivers events after `startOffset` and
   keeps its cursor in memory, writing no consumer-offset row; it is for
   subscribers that live as long as a connection (websocket fan-out, a CLI
   tail). The subscribe loop only saves offsets when it has a consumer name.
   The Python, JVM and .NET bindings already take a `from_offset` and skip the
   save when there is no consumer.

3. **Poll idle stream subscribers when the commit watcher dies.** When
   `honker_watcher_wait` returns an unexpected code the watcher loop exits for
   good and nothing wakes a subscriber again. The patch logs the exit with
   `slog.Error` and gives the subscriber's idle wait a 5-second fallback poll
   (`streamPollFallback`), with a test that kills the watcher under a live
   subscriber. Upstream fixed the same hang for node and bun in PRs #73, #74
   and #75.

4. This README (`.github/README.md`, which GitHub shows in place of the root
   README, so upstream's README never conflicts on a rebase).

## Testing

```
cargo build -p honker-extension --release    # or symlink a built extension to target/release/libhonker_ext.so
cd packages/honker-go && go test -race -count=1 ./...
```

The Go tests look for the extension under `target/{debug,release}/` and skip
when it is missing.

## Licence

Same as upstream: MIT or Apache-2.0, at your option (`LICENSE-MIT`,
`LICENSE-APACHE`).
