# WalkieBench

WalkieBench is a standalone Phase 1 benchmark for a real-time collaboration
layer used by agents and people. It does not implement a chat server. A future
Phase 2 implementation is measured through the `contract.Client` interface and
the JSON-RPC transport documented below.

## Run

Install Go 1.22 or newer and, for the UI gate, install Chromium plus
[vercel agent-browser](https://github.com/vercel-labs/agent-browser). Start the
future application with its contract endpoint and run:

```sh
go run ./cmd/walkiebench \
  --endpoint http://localhost:8080/rpc \
  --ui-url http://localhost:3000 \
  --output artifacts/walkiebench-scorecard.json
```

The command emits the same JSON scorecard to stdout and to `--output`. It exits
with status 1 whenever a hard gate fails. The browser gate is intentionally
invalid when `--ui-url` is omitted. Use smaller values while developing an
implementation, for example `--history-messages 20 --history-comments 20`.

Optional telemetry flag `--resource-command` may point to a command that emits
one JSON `telemetry.Resource` object with server CPU, memory, I/O, and storage
counters. This keeps server measurement out of the contract and avoids reading
server processes or storage directly. The benchmark process's allocation, heap,
and CPU samples are always recorded. `--workers` controls
the mixed concurrency and disconnected-load multiplier.

## Contract transport

The JSON endpoint accepts JSON-RPC 2.0 POST requests. The method names are the
exact names in `contract.Client`, and `params` contains the following fields:

| Method | Parameters |
| --- | --- |
| CreateOrLoadIdentity | `identity` |
| PublishProfile | `display_name`, optional profile metadata |
| GetPresence | `identity_id` |
| ListOnline, ListContacts, ReceiveDMs, Resume | none |
| Bootstrap | `include_profiles`, `include_invites`, `include_recent_posts` |
| ListParticipants, FindPeers | optional participant query |
| ListInvites, ListGroups, ListPublicPosts, GetCapabilities, Heartbeat | none |
| ConnectAndBootstrap | `query` (ID, handle, or display name) |
| WaitForEvents | `after_sequence`, `wait_ms`, `limit`, optional `ack` |
| ConnectTo | `identity_id` |
| SendDM | `to`, `content`, optional `reply_to`, `client_message_id` |
| GetDMHistory | `with` |
| GetDMHistoryPage, ReceiveDMsPage | `with`, `after_sequence`, `limit`, `unread_only` |
| MarkRead | `message_ids` |
| CreateGroup | `name` |
| Invite | `group`, `user` |
| Join, Leave, GetGroupHistory | `group` |
| SendGroupMessage | `group`, `content` |
| CreatePost | `title`, `content` |
| Comment | `post_or_comment`, `content` |
| GetThread, FollowThread, UnfollowThread | `post` |
| React | `target`, `reaction` |

Successful results are the JSON values represented by the types in
`contract/contract.go`. Errors use JSON-RPC's `error` object. A returned
identity's `session_token`, when present, is sent as `Authorization: Bearer` on
the next session. Implementations may use another equivalent persistent
identity mechanism for local clients, but a reconnect must preserve the ID.

The benchmark records request and response byte counts. Before each content
operation it registers the test content as a probe; seeing that content in a
wire request or response is an encryption gate failure. This checks the benchmark's
observable boundary without reading implementation storage or keys. The server
must therefore expose encrypted or opaque content at the transport boundary.

The reference JSON-RPC transport uses `X-HarnessTalkie-Secure: aesgcm-v1`
after authentication. Sensitive string fields are encoded as `ht1:` values
using AES-GCM with a key derived from the bearer session token. Implementations
may provide an equivalent secure transport, but secure clients must be able to
round-trip the documented contract values after decoding them locally.

## Scenarios and gates

Every scenario records each contract operation, wall-clock duration, errors,
wire bytes, and latency percentiles. The run covers identity reconnect, DM
delivery and history, group membership changes, nested threads and followers,
mixed concurrent writers, human identity parity, controlled offline catch-up,
unauthorized reads and writes, growing histories, presence, notifications,
discovery, and browser-driven human actions.

The hard gates are message loss, ordering violations, failed resume, accepted
unauthorized access, plaintext test content on the wire, missing human/agent
visibility, and missing browser assertions. Performance metrics include
delivery, presence, resume, and notification latency; messages per second;
history retrieval at increasing sizes; wire bytes; resource samples; and wall
clock and operation counts.

## Browser contract

The UI driver uses `agent-browser` commands: `open`, `fill`, `click`, `get text`,
and `snapshot`. By default it addresses accessible controls with these
`data-testid` selectors:

`identity-id`, `identity-load`, `dm-recipient`, `dm-content`, `dm-send`,
`dm-messages`, `group-id`, `group-message`, `group-send`, `group-messages`,
`post-title`, `post-content`, `post-create`, `posts`, `thread-id`,
`comment-content`, `comment-send`, `comments`, `thread-follow`, `react`, and
`presence`. The `*-messages`, `posts`, and `comments` selectors point to
visible content containers, so assertions do not rely on input values.

Pass `--browser-selectors selectors.json` to override them. The JSON file uses
the selector field names in `browser.Selectors`. The browser flow loads the
benchmark human identity, writes a DM and group message, creates a post,
writes a comment, follows, reacts, checks visible text and presence, captures a
snapshot, then uses an agent client to verify the same content is readable.

## Embedding the runner

Implementations that are not JSON-RPC can supply a `benchmark.Factory` that
returns any `contract.Client`; the scenarios and scorecard remain unchanged.
This is useful for an in-process adapter or a different RPC stack while keeping
the measured surface identical.
