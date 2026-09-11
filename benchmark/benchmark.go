// Package benchmark contains the scenario runner and all benchmark assertions.
package benchmark

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"walkiebench/browser"
	"walkiebench/contract"
	"walkiebench/telemetry"
)

type Factory interface {
	New(context.Context, *contract.Identity) (contract.Client, error)
}
type Config struct {
	Workers                int
	HistoryMessages        int
	HistoryComments        int
	MaxDeliveryLatency     time.Duration
	MaxPresenceLatency     time.Duration
	MaxResumeLatency       time.Duration
	MaxNotificationLatency time.Duration
	ResourceCommand        string
	BrowserURL             string
	BrowserCommand         string
	BrowserSelectors       browser.Selectors
	Profile                string
	Seed                   int64
	RequireV2              bool
}

func (c *Config) defaults() {
	if c.Workers < 1 {
		c.Workers = 3
	}
	if c.HistoryMessages < 1 {
		c.HistoryMessages = 2000
	}
	if c.HistoryComments < 1 {
		c.HistoryComments = 1000
	}
	if c.MaxDeliveryLatency == 0 {
		c.MaxDeliveryLatency = 5 * time.Second
	}
	if c.MaxPresenceLatency == 0 {
		c.MaxPresenceLatency = 5 * time.Second
	}
	if c.MaxResumeLatency == 0 {
		c.MaxResumeLatency = 10 * time.Second
	}
	if c.MaxNotificationLatency == 0 {
		c.MaxNotificationLatency = 5 * time.Second
	}
	if c.Profile == "" {
		c.Profile = "full"
	}
	if c.Seed == 0 {
		c.Seed = 20260910
	}
	if p, ok := Profile(c.Profile); ok && p.RequiresV2 {
		c.RequireV2 = true
	}
}

type Session struct {
	client    contract.Client
	identity  contract.Identity
	telemetry *telemetry.Collector
	v2        contract.V2Client
	discovery contract.DiscoveryClient
}

func newSession(c contract.Client, t *telemetry.Collector) *Session {
	s := &Session{client: c, telemetry: t}
	if v, ok := c.(contract.V2Client); ok {
		s.v2 = v
	}
	if d, ok := c.(contract.DiscoveryClient); ok {
		s.discovery = d
	}
	return s
}
func (s *Session) identityID() string { return s.identity.ID }
func call[T any](s *Session, name string, fn func() (T, error)) (T, error) {
	start := time.Now()
	v, e := fn()
	s.telemetry.Observe("", name, time.Since(start), e == nil, e)
	return v, e
}
func (s *Session) callErr(name string, fn func() error) error {
	start := time.Now()
	e := fn()
	s.telemetry.Observe("", name, time.Since(start), e == nil, e)
	return e
}
func (s *Session) CreateOrLoadIdentity(c context.Context, n string) (contract.Identity, error) {
	v, e := call(s, "CreateOrLoadIdentity", func() (contract.Identity, error) { return s.client.CreateOrLoadIdentity(c, n) })
	if e == nil {
		s.identity = v
	}
	return v, e
}
func (s *Session) PublishProfile(c context.Context, p contract.Profile) error {
	// DisplayName, Repository, and Harness are discoverable routing metadata.
	// Probe only profile fields that the secure-wire contract protects.
	for _, value := range []string{p.Bio, p.CurrentWork, p.Limitations} {
		s.telemetry.RegisterPlaintext(value)
	}
	for _, value := range append(append([]string{}, p.Capabilities...), p.CollaborationTopics...) {
		s.telemetry.RegisterPlaintext(value)
	}
	return s.callErr("PublishProfile", func() error { return s.client.PublishProfile(c, p) })
}
func (s *Session) GetPresence(c context.Context, id string) (contract.Presence, error) {
	return call(s, "GetPresence", func() (contract.Presence, error) { return s.client.GetPresence(c, id) })
}
func (s *Session) ListOnline(c context.Context) ([]contract.Presence, error) {
	return call(s, "ListOnline", func() ([]contract.Presence, error) { return s.client.ListOnline(c) })
}
func (s *Session) ConnectTo(c context.Context, id string) error {
	return s.callErr("ConnectTo", func() error { return s.client.ConnectTo(c, id) })
}
func (s *Session) ListContacts(c context.Context) ([]contract.Contact, error) {
	return call(s, "ListContacts", func() ([]contract.Contact, error) { return s.client.ListContacts(c) })
}
func (s *Session) ListParticipants(c context.Context, q contract.ParticipantQuery) ([]contract.Participant, error) {
	return call(s, "ListParticipants", func() ([]contract.Participant, error) { return s.client.ListParticipants(c, q) })
}
func (s *Session) FindPeers(c context.Context, q contract.ParticipantQuery) ([]contract.Participant, error) {
	return call(s, "FindPeers", func() ([]contract.Participant, error) { return s.client.FindPeers(c, q) })
}
func (s *Session) ListInvites(c context.Context) ([]contract.Invitation, error) {
	return call(s, "ListInvites", func() ([]contract.Invitation, error) { return s.client.ListInvites(c) })
}
func (s *Session) Heartbeat(c context.Context) error {
	return s.callErr("Heartbeat", func() error { return s.client.Heartbeat(c) })
}
func (s *Session) Bootstrap(c context.Context, profiles, invites, posts bool) (contract.Bootstrap, error) {
	return call(s, "Bootstrap", func() (contract.Bootstrap, error) { return s.client.Bootstrap(c, profiles, invites, posts) })
}
func (s *Session) ConnectAndBootstrap(c context.Context, query string) (contract.Bootstrap, error) {
	return call(s, "ConnectAndBootstrap", func() (contract.Bootstrap, error) { return s.client.ConnectAndBootstrap(c, query) })
}
func (s *Session) WaitForEvents(c context.Context, q contract.EventQuery) (contract.EventBatch, error) {
	return call(s, "WaitForEvents", func() (contract.EventBatch, error) { return s.client.WaitForEvents(c, q) })
}
func (s *Session) SendDM(c context.Context, to, m string) (contract.Message, error) {
	s.telemetry.RegisterPlaintext(m)
	return call(s, "SendDM", func() (contract.Message, error) { return s.client.SendDM(c, to, m) })
}
func (s *Session) SendDMWithOptions(c context.Context, req contract.SendDMRequest) (contract.Message, error) {
	s.telemetry.RegisterPlaintext(req.Content)
	return call(s, "SendDMWithOptions", func() (contract.Message, error) { return s.client.SendDMWithOptions(c, req) })
}
func (s *Session) GetDMHistory(c context.Context, id string) ([]contract.Message, error) {
	return call(s, "GetDMHistory", func() ([]contract.Message, error) { return s.client.GetDMHistory(c, id) })
}
func (s *Session) ReceiveDMs(c context.Context) ([]contract.Message, error) {
	return call(s, "ReceiveDMs", func() ([]contract.Message, error) { return s.client.ReceiveDMs(c) })
}
func (s *Session) MarkRead(c context.Context, ids []string) error {
	return s.callErr("MarkRead", func() error { return s.client.MarkRead(c, ids) })
}
func (s *Session) CreateGroup(c context.Context, n string) (contract.Group, error) {
	return call(s, "CreateGroup", func() (contract.Group, error) { return s.client.CreateGroup(c, n) })
}
func (s *Session) Invite(c context.Context, g, u string) error {
	return s.callErr("Invite", func() error { return s.client.Invite(c, g, u) })
}
func (s *Session) Join(c context.Context, g string) error {
	return s.callErr("Join", func() error { return s.client.Join(c, g) })
}
func (s *Session) Leave(c context.Context, g string) error {
	return s.callErr("Leave", func() error { return s.client.Leave(c, g) })
}
func (s *Session) SendGroupMessage(c context.Context, g, m string) (contract.Message, error) {
	s.telemetry.RegisterPlaintext(m)
	return call(s, "SendGroupMessage", func() (contract.Message, error) { return s.client.SendGroupMessage(c, g, m) })
}
func (s *Session) GetGroupHistory(c context.Context, g string) ([]contract.Message, error) {
	return call(s, "GetGroupHistory", func() ([]contract.Message, error) { return s.client.GetGroupHistory(c, g) })
}
func (s *Session) CreatePost(c context.Context, t, b string) (contract.Post, error) {
	s.telemetry.RegisterPlaintext(t)
	s.telemetry.RegisterPlaintext(b)
	return call(s, "CreatePost", func() (contract.Post, error) { return s.client.CreatePost(c, t, b) })
}
func (s *Session) Comment(c context.Context, p, b string) (contract.CommentNode, error) {
	s.telemetry.RegisterPlaintext(b)
	return call(s, "Comment", func() (contract.CommentNode, error) { return s.client.Comment(c, p, b) })
}
func (s *Session) GetThread(c context.Context, p string) (contract.Thread, error) {
	return call(s, "GetThread", func() (contract.Thread, error) { return s.client.GetThread(c, p) })
}
func (s *Session) FollowThread(c context.Context, p string) error {
	return s.callErr("FollowThread", func() error { return s.client.FollowThread(c, p) })
}
func (s *Session) UnfollowThread(c context.Context, p string) error {
	return s.callErr("UnfollowThread", func() error { return s.client.UnfollowThread(c, p) })
}
func (s *Session) React(c context.Context, t, r string) error {
	return s.callErr("React", func() error { return s.client.React(c, t, r) })
}
func (s *Session) Resume(c context.Context) (contract.ResumeResult, error) {
	return call(s, "Resume", func() (contract.ResumeResult, error) { return s.client.Resume(c) })
}
func (s *Session) Close() error { return s.client.Close() }

type Harness struct {
	factory    Factory
	cfg        Config
	t          *telemetry.Collector
	sessions   map[string]*Session
	identities map[string]contract.Identity
	groupID    string
	postID     string
	serverID   string
	groupIDV2  string
	postIDV2   string
	browser    browser.Driver
	rng        *rand.Rand
}

func New(factory Factory, cfg Config, t *telemetry.Collector, driver browser.Driver) *Harness {
	cfg.defaults()
	return &Harness{factory: factory, cfg: cfg, t: t, sessions: map[string]*Session{}, identities: map[string]contract.Identity{}, browser: driver, rng: rand.New(rand.NewSource(cfg.Seed))}
}

func (h *Harness) setup(ctx context.Context) error {
	for _, name := range []string{"agent-a", "agent-b", "agent-c", "human"} {
		raw, e := h.factory.New(ctx, nil)
		if e != nil {
			return fmt.Errorf("new %s session: %w", name, e)
		}
		s := newSession(raw, h.t)
		id, e := s.CreateOrLoadIdentity(ctx, name)
		if e != nil {
			return fmt.Errorf("identity %s: %w", name, e)
		}
		s.identity = id
		h.sessions[name] = s
		h.identities[name] = id
		kind := "agent"
		if name == "human" {
			kind = "human"
		}
		if e = s.PublishProfile(ctx, contract.Profile{DisplayName: name, Kind: kind, Bio: "WalkieBench participant", Repository: "WalkieBench", Harness: "benchmark", Capabilities: []string{"benchmarking", "collaboration"}, CurrentWork: "WalkieBench interoperability", CollaborationTopics: []string{"realtime systems", "agent coordination"}}); e != nil {
			return fmt.Errorf("profile %s: %w", name, e)
		}
	}
	return nil
}
func (h *Harness) reconnect(ctx context.Context, name string) (*Session, error) {
	old := h.identities[name]
	raw, e := h.factory.New(ctx, &old)
	if e != nil {
		return nil, e
	}
	s := newSession(raw, h.t)
	id, e := s.CreateOrLoadIdentity(ctx, old.ID)
	if e != nil {
		return nil, e
	}
	if id.ID != old.ID {
		return nil, fmt.Errorf("identity changed across reconnect: %s -> %s", old.ID, id.ID)
	}
	s.identity = id
	h.sessions[name] = s
	h.identities[name] = id
	return s, nil
}

func (h *Harness) Run(ctx context.Context) telemetry.Scorecard {
	started := time.Now()
	h.t.SetRunMetadata(h.cfg.Profile, h.cfg.Seed)
	if _, ok := Profile(h.cfg.Profile); !ok {
		h.t.AddInvalid("unknown benchmark profile: " + h.cfg.Profile)
		return h.t.Snapshot()
	}
	if e := h.setup(ctx); e != nil {
		h.t.AddInvalid(e.Error())
		return h.t.Snapshot()
	}
	stopResources := startResourceSampler(ctx, h.cfg, h.t)
	defer stopResources()
	scenarios := []struct {
		name     string
		category string
		hard     bool
		fn       func(context.Context) []error
	}{
		{"identity_lifecycle", "core", true, h.identityLifecycle}, {"discovery_bootstrap_events", "core", true, h.discoveryScenario}, {"dm_round_trip_resume", "core", true, h.dmScenario}, {"group_membership_history", "core", true, h.groupScenario}, {"post_nested_thread", "core", true, h.postScenario}, {"concurrent_mixed_writers", "load", true, h.concurrentScenario}, {"human_agent_parity", "core", true, h.humanParityScenario}, {"disconnect_reconnect_load", "reliability", true, h.loadScenario}, {"unauthorized_access", "security", true, h.unauthorizedScenario}, {"growing_history", "load", true, h.growingHistoryScenario}, {"presence_notifications", "core", true, h.presenceScenario},
		{"security_isolation_abuse", "security", true, h.securityIsolationScenario}, {"server_lifecycle", "server", true, h.serverLifecycleScenario}, {"server_permissions_discovery", "permissions", true, h.serverPermissionsScenario}, {"granular_permission_enforcement", "permissions", true, h.granularPermissionScenario}, {"server_groups", "groups", true, h.serverGroupsScenario}, {"server_forums_notifications", "forums", true, h.serverForumsScenario}, {"notification_durability_matrix", "forums", true, h.notificationDurabilityScenario}, {"declarative_batch", "declarative", true, h.declarativeScenario}, {"delta_response_shaping", "declarative", true, h.deltaScenario}, {"protocol_discovery_schemas", "transport", true, h.protocolDiscoveryScenario}, {"transport_interoperability", "transport", false, h.transportScenario}, {"agent_efficiency_jobs", "efficiency", true, h.agentEfficiencyScenario}, {"v2_scale_load", "load", true, h.v2ScaleScenario},
	}
	for _, sc := range scenarios {
		if !profileAllows(h.cfg.Profile, sc.category) {
			continue
		}
		h.runScenario(ctx, sc.name, sc.category, sc.hard, sc.fn)
	}
	if profileAllows(h.cfg.Profile, "browser") {
		if h.browser == nil || h.cfg.BrowserURL == "" {
			h.t.AddInvalid("browser scenario requires --ui-url and a Chromium driver")
			h.runScenario(ctx, "browser_human_ui", "browser", true, func(context.Context) []error {
				return []error{fmt.Errorf("browser scenario requires --ui-url and a Chromium driver")}
			})
		} else {
			h.runScenario(ctx, "browser_human_ui", "browser", true, h.browserScenario)
		}
	}
	h.t.Metric("run_wall_clock_ms", float64(time.Since(started).Microseconds())/1000)
	snapshot := h.t.Snapshot()
	h.t.SetEfficiencyScore(EfficiencyScore(snapshot))
	return h.t.Snapshot()
}

func (h *Harness) runScenario(ctx context.Context, name, category string, hard bool, fn func(context.Context) []error) {
	h.t.BeginScenario(name)
	errs := fn(ctx)
	status := "passed"
	if len(errs) > 0 {
		status = "failed"
		unsupported := true
		for _, e := range errs {
			var u UnsupportedError
			if !errors.As(e, &u) {
				unsupported = false
				break
			}
		}
		if unsupported {
			status = "unsupported"
			if hard && h.cfg.RequireV2 {
				h.t.AddInvalid(name + ": required capability unsupported")
			}
		}
	}
	h.t.EndScenarioStatus(status, hard, errs)
	categoryScore := 0.0
	if status == "passed" {
		categoryScore = 100
	}
	h.t.SetCategory(category, status == "passed", hard, categoryScore, status)
}
func (h *Harness) label(prefix string) string { return fmt.Sprintf("%s-%08x", prefix, h.rng.Uint32()) }

func (h *Harness) identityLifecycle(ctx context.Context) []error {
	a := h.sessions["agent-a"]
	if e := a.Close(); e != nil {
		return []error{e}
	}
	s, e := h.reconnect(ctx, "agent-a")
	if e != nil {
		return []error{e}
	}
	r, e := h.resumeChecked(ctx, s)
	if e != nil {
		return []error{e}
	}
	if r.RestoredMessages < 0 {
		return []error{fmt.Errorf("invalid resume message count")}
	}
	return nil
}

func (h *Harness) discoveryScenario(ctx context.Context) []error {
	a, b, c := h.sessions["agent-a"], h.sessions["agent-b"], h.sessions["agent-c"]
	started := time.Now()
	boot, e := a.Bootstrap(ctx, true, true, true)
	if e != nil {
		return []error{e}
	}
	h.t.Metric("bootstrap_latency_ms", float64(time.Since(started).Microseconds())/1000)
	if boot.Identity.ID != a.identityID() || !hasParticipant(boot.Participants, b.identityID()) {
		return []error{fmt.Errorf("bootstrap omitted caller or known participant")}
	}
	peers, e := a.FindPeers(ctx, contract.ParticipantQuery{Capability: "benchmarking"})
	if e != nil {
		return []error{e}
	}
	if !hasParticipant(peers, b.identityID()) || peers[0].Repository == "" || peers[0].Harness == "" {
		return []error{fmt.Errorf("FindPeers omitted profile metadata")}
	}
	if e = a.Heartbeat(ctx); e != nil {
		return []error{e}
	}
	connected, e := a.ConnectAndBootstrap(ctx, b.identityID())
	if e != nil {
		return []error{e}
	}
	if !hasParticipant(connected.Participants, b.identityID()) {
		return []error{fmt.Errorf("ConnectAndBootstrap omitted peer")}
	}
	g, e := a.CreateGroup(ctx, "discovery-invites")
	if e != nil {
		return []error{e}
	}
	if e = a.Invite(ctx, g.ID, c.identityID()); e != nil {
		return []error{e}
	}
	invites, e := c.ListInvites(ctx)
	if e != nil {
		return []error{e}
	}
	if !hasInvitation(invites, g.ID) {
		return []error{fmt.Errorf("ListInvites omitted pending invitation")}
	}
	cursor := boot.Cursor
	messageID := fmt.Sprintf("discovery-%d", time.Now().UnixNano())
	first, e := a.SendDMWithOptions(ctx, contract.SendDMRequest{To: b.identityID(), Content: "cursor-discovery", ClientMessageID: messageID})
	if e != nil {
		return []error{e}
	}
	second, e := a.SendDMWithOptions(ctx, contract.SendDMRequest{To: b.identityID(), Content: "cursor-discovery", ClientMessageID: messageID})
	if e != nil {
		return []error{e}
	}
	if first.ID != second.ID {
		return []error{fmt.Errorf("idempotent SendDM returned different message IDs")}
	}
	batch, e := b.WaitForEvents(ctx, contract.EventQuery{AfterSequence: cursor, WaitMS: 1000, Limit: 20, Ack: true})
	if e != nil {
		return []error{e}
	}
	found := false
	for _, event := range batch.Events {
		if event.Message != nil && event.Message.ID == first.ID && event.Message.Content == "cursor-discovery" {
			found = true
		}
	}
	if !found {
		return []error{fmt.Errorf("WaitForEvents omitted new DM")}
	}
	return nil
}

func (h *Harness) dmScenario(ctx context.Context) []error {
	a, b := h.sessions["agent-a"], h.sessions["agent-b"]
	body := "dm-round-trip-" + fmt.Sprint(time.Now().UnixNano())
	start := time.Now()
	if _, e := a.SendDM(ctx, b.identityID(), body); e != nil {
		return []error{e}
	}
	var got []contract.Message
	var e error
	for time.Since(start) < h.cfg.MaxDeliveryLatency {
		got, e = b.ReceiveDMs(ctx)
		if e != nil {
			return []error{e}
		}
		if hasContent(got, body) {
			h.t.Metric("dm_delivery_latency_ms", float64(time.Since(start).Microseconds())/1000)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !hasContent(got, body) {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.MessageLoss++ })
		return []error{fmt.Errorf("DM was not delivered")}
	}
	hist, e := b.GetDMHistory(ctx, a.identityID())
	if e != nil {
		return []error{e}
	}
	if !hasContent(hist, body) {
		return []error{fmt.Errorf("DM missing from history")}
	}
	if es := assertOrdered(hist, h.t); len(es) > 0 {
		return es
	}
	ids := messageIDs(got, body)
	if e = b.MarkRead(ctx, ids); e != nil {
		return []error{e}
	}
	if e = a.Close(); e != nil {
		return []error{e}
	}
	if _, e = h.reconnect(ctx, "agent-a"); e != nil {
		return []error{e}
	}
	if _, e = h.reconnect(ctx, "agent-b"); e != nil {
		return []error{e}
	}
	r, e := h.sessions["agent-b"].Resume(ctx)
	if e != nil {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.FailedResumes++ })
		return []error{e}
	}
	if r.RestoredMessages < 0 {
		return []error{fmt.Errorf("invalid DM resume result")}
	}
	hist, e = h.sessions["agent-b"].GetDMHistory(ctx, a.identityID())
	if e != nil || !hasContent(hist, body) {
		if e == nil {
			e = fmt.Errorf("DM history missing after resume")
		}
		return []error{e}
	}
	return nil
}

func (h *Harness) groupScenario(ctx context.Context) []error {
	a, b, c := h.sessions["agent-a"], h.sessions["agent-b"], h.sessions["agent-c"]
	g, e := a.CreateGroup(ctx, "membership-gate")
	if e != nil {
		return []error{e}
	}
	h.groupID = g.ID
	for _, id := range []string{b.identityID(), c.identityID()} {
		if e = a.Invite(ctx, g.ID, id); e != nil {
			return []error{e}
		}
	}
	if e = b.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	if e = c.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	for _, x := range []struct {
		s *Session
		v string
	}{{a, "group-a"}, {b, "group-b"}, {c, "group-c"}} {
		if _, e = x.s.SendGroupMessage(ctx, g.ID, x.v); e != nil {
			return []error{e}
		}
	}
	hist, e := b.GetGroupHistory(ctx, g.ID)
	if e != nil {
		return []error{e}
	}
	if len(hist) != 3 {
		return []error{fmt.Errorf("expected 3 group messages, got %d", len(hist))}
	}
	if e = c.Leave(ctx, g.ID); e != nil {
		return []error{e}
	}
	_, e = c.SendGroupMessage(ctx, g.ID, "after-leave")
	if e == nil {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.AccessControlViolations++ })
		return []error{fmt.Errorf("former member could write after leaving")}
	}
	if _, e = c.GetGroupHistory(ctx, g.ID); e == nil {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.AccessControlViolations++ })
		return []error{fmt.Errorf("former member could read after leaving")}
	}
	return assertOrdered(hist, h.t)
}

func (h *Harness) postScenario(ctx context.Context) []error {
	a, b, c := h.sessions["agent-a"], h.sessions["agent-b"], h.sessions["agent-c"]
	p, e := a.CreatePost(ctx, "thread-benchmark", "root-post")
	if e != nil {
		return []error{e}
	}
	h.postID = p.ID
	if e = b.FollowThread(ctx, p.ID); e != nil {
		return []error{e}
	}
	root, e := b.Comment(ctx, p.ID, "root-comment")
	if e != nil {
		return []error{e}
	}
	nested, e := c.Comment(ctx, root.ID, "nested-comment")
	if e != nil {
		return []error{e}
	}
	if nested.ParentID != root.ID {
		return []error{fmt.Errorf("nested comment parent mismatch")}
	}
	if e = a.React(ctx, p.ID, "like"); e != nil {
		return []error{e}
	}
	th, e := b.GetThread(ctx, p.ID)
	if e != nil {
		return []error{e}
	}
	if countComments(th.Comments) != 2 {
		return []error{fmt.Errorf("expected nested comment tree with 2 comments, got %d", countComments(th.Comments))}
	}
	if e := assertCommentsOrdered(th.Comments, h.t); e != nil {
		return []error{e}
	}
	if _, e = a.Comment(ctx, p.ID, "follower-update"); e != nil {
		return []error{e}
	}
	start := time.Now()
	for time.Since(start) < h.cfg.MaxNotificationLatency {
		th, e = b.GetThread(ctx, p.ID)
		if e != nil {
			return []error{e}
		}
		if countComments(th.Comments) >= 3 {
			h.t.Metric("thread_notification_latency_ms", float64(time.Since(start).Microseconds())/1000)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if countComments(th.Comments) < 3 {
		return []error{fmt.Errorf("follower did not receive thread update")}
	}
	if e = b.UnfollowThread(ctx, p.ID); e != nil {
		return []error{e}
	}
	if e = b.Close(); e != nil {
		return []error{e}
	}
	if _, e = h.reconnect(ctx, "agent-b"); e != nil {
		return []error{e}
	}
	if _, e = h.resumeChecked(ctx, h.sessions["agent-b"]); e != nil {
		return []error{e}
	}
	th, e = h.sessions["agent-b"].GetThread(ctx, p.ID)
	if e != nil || countComments(th.Comments) != 3 {
		if e == nil {
			e = fmt.Errorf("thread history missing after resume")
		}
		return []error{e}
	}
	return nil
}

func (h *Harness) concurrentScenario(ctx context.Context) []error {
	a, b, c := h.sessions["agent-a"], h.sessions["agent-b"], h.sessions["agent-c"]
	g, e := a.CreateGroup(ctx, "concurrent-writers")
	if e != nil {
		return []error{e}
	}
	for _, id := range []string{b.identityID(), c.identityID()} {
		if e = a.Invite(ctx, g.ID, id); e != nil {
			return []error{e}
		}
	}
	if e = b.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	if e = c.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	p, e := a.CreatePost(ctx, "concurrent-thread", "thread-content-probe")
	if e != nil {
		return []error{e}
	}
	writers := []*Session{a, b, c}
	h.t.Metric("max_concurrent_participants", float64(len(writers)))
	mixedStart := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for i, s := range writers {
		wg.Add(1)
		go func(i int, s *Session) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if _, x := s.SendGroupMessage(ctx, g.ID, fmt.Sprintf("concurrent-group-%d-%d", i, j)); x != nil {
					mu.Lock()
					errs = append(errs, x)
					mu.Unlock()
				}
				if _, x := s.Comment(ctx, p.ID, fmt.Sprintf("concurrent-comment-%d-%d", i, j)); x != nil {
					mu.Lock()
					errs = append(errs, x)
					mu.Unlock()
				}
			}
		}(i, s)
	}
	wg.Wait()
	h.t.Metric("peak_mixed_messages_per_second", 120/time.Since(mixedStart).Seconds())
	if len(errs) > 0 {
		return errs
	}
	hist, e := a.GetGroupHistory(ctx, g.ID)
	if e != nil {
		return []error{e}
	}
	if len(hist) != 60 {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.MessageLoss += 60 - len(hist) })
		return []error{fmt.Errorf("concurrent group loss: expected 60, got %d", len(hist))}
	}
	if es := assertOrdered(hist, h.t); len(es) > 0 {
		return es
	}
	th, e := b.GetThread(ctx, p.ID)
	if e != nil {
		return []error{e}
	}
	if countComments(th.Comments) != 60 {
		return []error{fmt.Errorf("concurrent thread loss: expected 60, got %d", countComments(th.Comments))}
	}
	if e := assertCommentsOrdered(th.Comments, h.t); e != nil {
		return []error{e}
	}
	return nil
}

func (h *Harness) humanParityScenario(ctx context.Context) []error {
	a, b, hu := h.sessions["agent-a"], h.sessions["agent-b"], h.sessions["human"]
	g, e := a.CreateGroup(ctx, "human-parity")
	if e != nil {
		return []error{e}
	}
	h.groupID = g.ID
	for _, id := range []string{b.identityID(), hu.identityID()} {
		if e = a.Invite(ctx, g.ID, id); e != nil {
			return []error{e}
		}
	}
	if e = b.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	if e = hu.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	p, e := a.CreatePost(ctx, "human-parity-thread", "agent-visible-post")
	if e != nil {
		return []error{e}
	}
	h.postID = p.ID
	if _, e = b.Comment(ctx, p.ID, "agent-visible-comment"); e != nil {
		return []error{e}
	}
	if _, e = hu.SendDM(ctx, a.identityID(), "human-client-dm"); e != nil {
		return []error{e}
	}
	if _, e = hu.SendGroupMessage(ctx, g.ID, "human-client-group"); e != nil {
		return []error{e}
	}
	humanPost, e := hu.CreatePost(ctx, "human-client-post", "human-client-post-content")
	if e != nil {
		return []error{e}
	}
	if _, e = a.GetThread(ctx, humanPost.ID); e != nil {
		return []error{fmt.Errorf("agent cannot read human post: %w", e)}
	}
	if _, e = hu.Comment(ctx, p.ID, "human-client-comment"); e != nil {
		return []error{e}
	}
	if e = hu.FollowThread(ctx, p.ID); e != nil {
		return []error{e}
	}
	if e = hu.React(ctx, p.ID, "like"); e != nil {
		return []error{e}
	}
	dm, e := a.GetDMHistory(ctx, hu.identityID())
	if e != nil || !hasContent(dm, "human-client-dm") {
		if e == nil {
			e = fmt.Errorf("agent cannot read human DM")
		}
		return []error{e}
	}
	gh, e := a.GetGroupHistory(ctx, g.ID)
	if e != nil || !hasContent(gh, "human-client-group") {
		if e == nil {
			e = fmt.Errorf("agent cannot read human group message")
		}
		return []error{e}
	}
	th, e := b.GetThread(ctx, p.ID)
	if e != nil || !hasContentComment(th.Comments, "human-client-comment") {
		if e == nil {
			e = fmt.Errorf("agent cannot read human comment")
		}
		return []error{e}
	}
	return nil
}

func (h *Harness) loadScenario(ctx context.Context) []error {
	a, b := h.sessions["agent-a"], h.sessions["agent-b"]
	g, e := a.CreateGroup(ctx, "disconnect-load")
	if e != nil {
		return []error{e}
	}
	if e = a.Invite(ctx, g.ID, b.identityID()); e != nil {
		return []error{e}
	}
	if e = b.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	if e = b.Close(); e != nil {
		return []error{e}
	}
	want := h.cfg.Workers * 100
	for i := 0; i < want; i++ {
		if _, e = a.SendGroupMessage(ctx, g.ID, fmt.Sprintf("offline-load-%05d", i)); e != nil {
			return []error{e}
		}
	}
	start := time.Now()
	if _, e = h.reconnect(ctx, "agent-b"); e != nil {
		return []error{e}
	}
	if _, e = h.sessions["agent-b"].Resume(ctx); e != nil {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.FailedResumes++ })
		return []error{e}
	}
	hist, e := h.sessions["agent-b"].GetGroupHistory(ctx, g.ID)
	if e != nil {
		return []error{e}
	}
	h.t.Metric("load_resume_latency_ms", float64(time.Since(start).Microseconds())/1000)
	if len(hist) != want {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.MessageLoss += want - len(hist) })
		return []error{fmt.Errorf("offline loss: expected %d, got %d", want, len(hist))}
	}
	return assertOrdered(hist, h.t)
}

func (h *Harness) unauthorizedScenario(ctx context.Context) []error {
	a, b, c := h.sessions["agent-a"], h.sessions["agent-b"], h.sessions["agent-c"]
	g, e := a.CreateGroup(ctx, "acl-gate")
	if e != nil {
		return []error{e}
	}
	if e = a.Invite(ctx, g.ID, b.identityID()); e != nil {
		return []error{e}
	}
	if e = b.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	if _, e = c.GetGroupHistory(ctx, g.ID); e == nil {
		return []error{fmt.Errorf("unauthorized group read accepted")}
	}
	if _, e = c.SendGroupMessage(ctx, g.ID, "unauthorized"); e == nil {
		return []error{fmt.Errorf("unauthorized group write accepted")}
	}
	if _, e = a.SendDM(ctx, b.identityID(), "private-probe"); e != nil {
		return []error{e}
	}
	if _, e = c.GetDMHistory(ctx, b.identityID()); e == nil {
		return []error{fmt.Errorf("unauthorized DM read accepted")}
	}
	return nil
}

func (h *Harness) growingHistoryScenario(ctx context.Context) []error {
	a, b := h.sessions["agent-a"], h.sessions["agent-b"]
	g, e := a.CreateGroup(ctx, "growing-history")
	if e != nil {
		return []error{e}
	}
	if e = a.Invite(ctx, g.ID, b.identityID()); e != nil {
		return []error{e}
	}
	if e = b.Join(ctx, g.ID); e != nil {
		return []error{e}
	}
	p, e := a.CreatePost(ctx, "growing-thread", "history")
	if e != nil {
		return []error{e}
	}
	last := 0
	sendStart := time.Now()
	for i := 0; i < h.cfg.HistoryMessages; i++ {
		if _, e = a.SendGroupMessage(ctx, g.ID, fmt.Sprintf("history-message-%06d", i)); e != nil {
			return []error{e}
		}
		if i%500 == 499 || i == h.cfg.HistoryMessages-1 {
			start := time.Now()
			hist, x := b.GetGroupHistory(ctx, g.ID)
			if x != nil {
				return []error{x}
			}
			h.t.Metric(fmt.Sprintf("group_history_%d_latency_ms", i+1), float64(time.Since(start).Microseconds())/1000)
			if len(hist) != i+1 {
				return []error{fmt.Errorf("history size %d returned %d", i+1, len(hist))}
			}
			last = i + 1
		}
	}
	h.t.Metric("sustained_messages_per_second", float64(h.cfg.HistoryMessages)/time.Since(sendStart).Seconds())
	for i := 0; i < h.cfg.HistoryComments; i++ {
		if _, e = a.Comment(ctx, p.ID, fmt.Sprintf("history-comment-%06d", i)); e != nil {
			return []error{e}
		}
		if i%500 == 499 || i == h.cfg.HistoryComments-1 {
			start := time.Now()
			th, x := b.GetThread(ctx, p.ID)
			if x != nil {
				return []error{x}
			}
			h.t.Metric(fmt.Sprintf("thread_history_%d_latency_ms", i+1), float64(time.Since(start).Microseconds())/1000)
			if countComments(th.Comments) != i+1 {
				return []error{fmt.Errorf("thread size %d returned %d", i+1, countComments(th.Comments))}
			}
			if x := assertCommentsOrdered(th.Comments, h.t); x != nil {
				return []error{x}
			}
		}
	}
	h.t.Metric("history_final_messages", float64(last))
	h.t.Metric("history_final_comments", float64(h.cfg.HistoryComments))
	return nil
}

func (h *Harness) presenceScenario(ctx context.Context) []error {
	a, b := h.sessions["agent-a"], h.sessions["agent-b"]
	if e := a.ConnectTo(ctx, b.identityID()); e != nil {
		return []error{e}
	}
	contacts, e := a.ListContacts(ctx)
	if e != nil {
		return []error{e}
	}
	if !hasContact(contacts, b.identityID()) {
		return []error{fmt.Errorf("ConnectTo did not update contacts")}
	}
	online, e := a.ListOnline(ctx)
	if e != nil {
		return []error{e}
	}
	if !hasPresence(online, b.identityID()) {
		return []error{fmt.Errorf("ListOnline omitted connected participant")}
	}
	if e = b.Close(); e != nil {
		return []error{e}
	}
	start := time.Now()
	for time.Since(start) < h.cfg.MaxPresenceLatency {
		p, x := a.GetPresence(ctx, b.identityID())
		if x != nil {
			return []error{x}
		}
		if !p.Online {
			h.t.Metric("presence_offline_latency_ms", float64(time.Since(start).Microseconds())/1000)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	p, e := a.GetPresence(ctx, b.identityID())
	if e != nil || p.Online {
		if e == nil {
			e = fmt.Errorf("offline presence not observed")
		}
		return []error{e}
	}
	if _, e = h.reconnect(ctx, "agent-b"); e != nil {
		return []error{e}
	}
	start = time.Now()
	for time.Since(start) < h.cfg.MaxPresenceLatency {
		p, e = a.GetPresence(ctx, b.identityID())
		if e != nil {
			return []error{e}
		}
		if p.Online {
			h.t.Metric("presence_online_latency_ms", float64(time.Since(start).Microseconds())/1000)
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return []error{fmt.Errorf("online presence not observed")}
}

func (h *Harness) browserScenario(ctx context.Context) []error {
	a, hu := h.sessions["agent-a"], h.sessions["human"]
	humanV2, err := h.v2("human")
	if err != nil {
		return []error{err}
	}
	adminServer, err := humanV2.v2.CreateServer(ctx, h.serverSpec(h.label("browser-admin"), contract.JoinApproval))
	if err != nil {
		return []error{err}
	}
	request, err := a.v2.RequestServerAccess(ctx, adminServer.ID, "browser approval workflow")
	if err != nil {
		return []error{err}
	}
	if e := hu.Close(); e != nil {
		return []error{e}
	}
	if _, e := browser.RunHumanFlow(ctx, h.browser, browser.Config{URL: h.cfg.BrowserURL, Selectors: h.cfg.BrowserSelectors, IdentityID: hu.identityID(), RequireApplicationIA: true, Observer: func(name string, d time.Duration, e error) { h.t.Observe("", "Browser."+name, d, e == nil, e) }}, h.groupID, h.postID, a.identityID()); e != nil {
		return []error{e}
	}
	if h.cfg.BrowserSelectors.ServerVisible != "" {
		if _, e := browser.RunAdminFlow(ctx, h.browser, browser.Config{URL: h.cfg.BrowserURL, Selectors: h.cfg.BrowserSelectors, IdentityID: hu.identityID(), RoleParticipant: a.identityID(), Observer: func(name string, d time.Duration, e error) { h.t.Observe("", "BrowserAdmin."+name, d, e == nil, e) }}, adminServer.ID); e != nil {
			return []error{e}
		}
		member, e := humanV2.v2.GetServerMember(ctx, adminServer.ID, a.identityID(), contract.ResponseOptions{})
		if e != nil || member.Role != contract.RoleModerator {
			if e == nil {
				e = fmt.Errorf("browser administration did not approve the request and assign moderator role")
			}
			return []error{e}
		}
		requests, e := humanV2.v2.ListServerRequests(ctx, adminServer.ID, contract.ResponseOptions{})
		if e != nil || !containsApprovedRequest(requests, request.ID) {
			if e == nil {
				e = fmt.Errorf("browser approval did not persist request lifecycle")
			}
			return []error{e}
		}
	}
	dm, e := a.GetDMHistory(ctx, hu.identityID())
	if e != nil || !hasContent(dm, "human-ui-dm") {
		if e == nil {
			e = fmt.Errorf("agent cannot read browser human DM")
		}
		return []error{e}
	}
	gh, e := a.GetGroupHistory(ctx, h.groupID)
	if e != nil || !hasContent(gh, "human-ui-group") {
		if e == nil {
			e = fmt.Errorf("agent cannot read browser human group message")
		}
		return []error{e}
	}
	th, e := a.GetThread(ctx, h.postID)
	if e != nil || !hasContentComment(th.Comments, "human-ui-comment") {
		if e == nil {
			e = fmt.Errorf("agent cannot read browser human comment")
		}
		return []error{e}
	}
	return nil
}

func (h *Harness) resumeChecked(ctx context.Context, s *Session) (contract.ResumeResult, error) {
	start := time.Now()
	r, e := s.Resume(ctx)
	h.t.Metric("resume_latency_ms", float64(time.Since(start).Microseconds())/1000)
	if e != nil {
		h.t.AddReliability(func(x *telemetry.Reliability) { x.FailedResumes++ })
		return r, e
	}
	if time.Since(start) > h.cfg.MaxResumeLatency {
		return r, fmt.Errorf("resume exceeded %s", h.cfg.MaxResumeLatency)
	}
	return r, nil
}

func hasContent(ms []contract.Message, s string) bool {
	for _, m := range ms {
		if m.Content == s {
			return true
		}
	}
	return false
}
func messageIDs(ms []contract.Message, s string) []string {
	var r []string
	for _, m := range ms {
		if m.Content == s {
			r = append(r, m.ID)
		}
	}
	return r
}
func hasContact(cs []contract.Contact, id string) bool {
	for _, c := range cs {
		if c.IdentityID == id {
			return true
		}
	}
	return false
}
func containsApprovedRequest(requests []contract.ServerAccessRequest, id string) bool {
	for _, request := range requests {
		if request.ID == id && request.Status == "approved" {
			return true
		}
	}
	return false
}
func hasParticipant(ps []contract.Participant, id string) bool {
	for _, p := range ps {
		if p.IdentityID == id {
			return true
		}
	}
	return false
}
func hasInvitation(xs []contract.Invitation, groupID string) bool {
	for _, x := range xs {
		if x.Group.ID == groupID {
			return true
		}
	}
	return false
}
func hasPresence(ps []contract.Presence, id string) bool {
	for _, p := range ps {
		if p.IdentityID == id && p.Online {
			return true
		}
	}
	return false
}
func assertOrdered(ms []contract.Message, t *telemetry.Collector) []error {
	for i := 1; i < len(ms); i++ {
		if ms[i].Sequence > 0 && ms[i-1].Sequence > 0 && ms[i].Sequence <= ms[i-1].Sequence {
			t.AddReliability(func(x *telemetry.Reliability) { x.OrderingViolations++ })
			return []error{fmt.Errorf("message ordering violation at %d", i)}
		}
	}
	return nil
}
func assertCommentsOrdered(ns []*contract.CommentNode, t *telemetry.Collector) error {
	for i := 1; i < len(ns); i++ {
		if !ns[i-1].CreatedAt.Before(ns[i].CreatedAt) {
			t.AddReliability(func(x *telemetry.Reliability) { x.OrderingViolations++ })
			return fmt.Errorf("comment ordering violation at %d", i)
		}
	}
	for _, n := range ns {
		if e := assertCommentsOrdered(n.Children, t); e != nil {
			return e
		}
	}
	return nil
}
func countComments(ns []*contract.CommentNode) int {
	n := 0
	for _, x := range ns {
		n++
		n += countComments(x.Children)
	}
	return n
}
func hasContentComment(ns []*contract.CommentNode, s string) bool {
	for _, x := range ns {
		if x.Content == s || hasContentComment(x.Children, s) {
			return true
		}
	}
	return false
}
