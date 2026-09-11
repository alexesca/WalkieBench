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

## V2 contract

WalkieBench V2 adds an optional `contract.V2Client` surface without changing
the V1 `contract.Client`. The surface is observable and implementation
independent: it does not prescribe storage, encryption architecture, language,
broker, database, frontend, or topology.

The Server operations are `CreateServer`, `GetServer`, `UpdateServer`,
`ListServers`, `DiscoverServers`, `JoinServer`, `RequestServerAccess`,
`ApproveServerRequest`, `RejectServerRequest`, `InviteToServer`,
`AcceptServerInvite`, `LeaveServer`, `RemoveServerMember`,
`ListServerMembers`, `GetServerMember`, `ListServerRequests`,
`ListServerInvites`, `FindServerMembers`, `ListServerRoles`,
`SetServerRole`, `UpdateServerPermissions`, and `GetServerAudit`.
Enrollment policies are `public`, `approval-required`, `invite-only`, and
`closed`. Server-private state and member directories must be inaccessible
until authorization is complete. Owner, administrator, moderator, member,
guest, and agent are stable role vocabulary; implementations may expose a
different internal role model if behavior is equivalent.

Server-scoped group operations are `CreateGroup`, `UpdateGroup`,
`DeleteGroup`, `DiscoverGroups`, `ListGroups`, `JoinGroup`,
`RequestGroupAccess`, `ApproveGroupRequest`, `RejectGroupRequest`,
`InviteToGroup`, `AcceptGroupInvite`, `LeaveGroup`, `RemoveGroupMember`, and
`ListGroupMembers`. Group policies are `public`, `approval-required`,
`invite-only`, and `private`/closed. Forum operations are `CreatePost`,
`EditPost`, `Comment`, `GetThread`, `DiscoverPosts`, `SearchPosts`,
`SharePost`, `ListNotifications`, and `MarkNotificationsRead`. They cover
server-wide, group-only, directly shared, and private visibility, nested
replies, mentions, follows, reactions, and notification recovery.

`ApplyManifest` is declarative session bootstrap. It is idempotent for desired
state: applying it twice cannot duplicate membership, requests, invitations,
contacts, subscriptions, or messages. `Batch` accepts ordered operations with
dependency IDs and returns deterministic per-operation results, including
partial failures. `Sync` accepts a server cursor and returns only new
authorized delta state. `ResponseOptions` supports field selection, compact
mode, limits, pagination, unread filtering, and cursor-based reads. Primitive
operations remain available for precise control.

`DiscoveryClient` exposes `DiscoverProtocol`, `GetSchema`, `GetHelp`,
`ListPresets`, `ApplyPreset`, and `ListTransports`. A discovery document must
identify protocol/version, capabilities, operations, transports, schema and
documentation locations, authentication requirements, and usage hints.
Schemas must be machine-readable and consistent with actual behavior. The
transport capability list allows optional HTTP, streaming HTTP, WebSocket, MCP,
CLI/stdio, local socket, filesystem, or future adapters. Implementations
advertising multiple read/write transports are tested for identity, membership,
message, history, cursor, and authorization parity.

## Profiles, metrics, and scoring

Use `--profile` to run a focused subset: `core`, `security`, `server`,
`permissions`, `groups`, `forums`, `agent-efficiency`, `declarative`,
`transport`, `browser`, `reliability`, `load`, or `full`. `full` requires V2;
focused V1 profiles remain useful against older implementations. Use `--seed`
for reproducible randomized names and content. V2 profiles report an explicit
`unsupported` scenario when a client lacks V2; required profiles mark that
absence invalid instead of silently skipping it.

Each scorecard contains raw operations, wire samples, percentile summaries,
resource samples, scenarios, category outcomes, reliability counters, and
metrics. Hard gates include zero message loss, ordering violations, failed
required resumes, accepted unauthorized access, invalid membership transitions,
cross-Server or cross-group leakage, duplicate idempotent state, plaintext
content on a protected contract wire, and failed required browser assertions.
A failed hard gate invalidates the run regardless of speed.

Agent Efficiency records operations, round trips, request/response/total wire
bytes, agent-visible bytes, deterministic token estimates, retries,
maintenance operations, successful actions, and time to first collaboration.
Token estimates use `(serialized_bytes + 3) / 4`, a stable proxy that needs no
online tokenizer. The bounded efficiency score is zero unless all gates pass,
then is:

```text
100 * (0.30 * min(1, 40 / operations)
     + 0.20 * min(1, 10 / round_trips)
     + 0.25 * min(1, 4000 / estimated_total_tokens)
     + 0.25 * min(1, 5000 / time_to_first_collaboration_ms))
```

The flagship onboarding job starts with a connection target, discovers the
protocol, joins a public Server, finds a participant by capability, sends a
DM, and verifies delivery. Primitive, batch, and manifest paths are compared
by correctness first, then operations, round trips, bytes, tokens, retries,
and elapsed time. The JSON scorecard is suitable for machine comparison after
every run.

## Scenarios and gates

Every scenario records each contract operation, wall-clock duration, errors,
wire bytes, and latency percentiles. The run covers identity reconnect, DM
delivery and history, group membership changes, nested threads and followers,
mixed concurrent writers, human identity parity, controlled offline catch-up,
unauthorized reads and writes, growing histories, presence, notifications,
discovery, Server policies and roles, server-scoped groups and forums,
mentions and notifications, manifests, batches, cursor deltas, response
shaping, transport discovery, agent onboarding, scale, and browser-driven
human actions.

The hard gates are message loss, ordering violations, failed resume, accepted
unauthorized access, plaintext test content on the wire, missing human/agent
visibility, and missing browser assertions. V2 adds Server policy, role,
isolation, manifest, batch, delta, discovery/schema, and transport gates.
Performance metrics include
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
`presence`. Optional `thread-visible`, `thread-structure`,
`notifications-visible`, and `ordered-activity-visible` selectors add visible
thread, notification, and ordering assertions. The `*-messages`, `posts`, and
`comments` selectors point to visible content containers, so assertions do not
rely on input values.

Pass `--browser-selectors selectors.json` to override them. The JSON file uses
the selector field names in `browser.Selectors`. The browser flow loads the
benchmark human identity, writes a DM and group message, creates a post,
writes a comment, follows, reacts, checks visible text and presence, captures a
snapshot, then uses an agent client to verify the same content is readable.
When V2 administration selectors are configured it also visibly checks Server,
member, request, group, moderation, and audit surfaces and exercises approval
and role controls. The additional selector names are `server-id`,
`server-visible`, `server-members-visible`, `server-requests-visible`,
`server-approve-request`, `server-role-participant`, `server-role-value`,
`server-role-save`, `group-admin-visible`, `moderation-visible`, and
`audit-visible`.

## Embedding the runner

Implementations that are not JSON-RPC can supply a `benchmark.Factory` that
returns any `contract.Client`; the scenarios and scorecard remain unchanged.
This is useful for an in-process adapter or a different RPC stack while keeping
the measured surface identical.
