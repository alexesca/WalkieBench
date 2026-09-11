package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"walkiebench/contract"
)

// UnsupportedError is returned by the benchmark when an optional client does
// not advertise the V2 capability needed by a scenario. This keeps V1 runs
// diagnosable while allowing profiles to make V2 support a hard gate.
type UnsupportedError struct{ Capability string }

func (e UnsupportedError) Error() string { return "unsupported contract capability: " + e.Capability }

func (h *Harness) v2(name string) (*Session, error) {
	s := h.sessions[name]
	if s == nil || s.v2 == nil {
		return nil, UnsupportedError{Capability: "V2Client"}
	}
	return s, nil
}

func (h *Harness) discovery(name string) (*Session, error) {
	s := h.sessions[name]
	if s == nil || s.discovery == nil {
		return nil, UnsupportedError{Capability: "DiscoveryClient"}
	}
	return s, nil
}

func v2Call[T any](s *Session, name string, f func() (T, error)) (T, error) {
	start := time.Now()
	v, err := f()
	s.telemetry.Observe("", "V2."+name, time.Since(start), err == nil, err)
	return v, err
}

func v2Err(s *Session, name string, f func() error) error {
	start := time.Now()
	err := f()
	s.telemetry.Observe("", "V2."+name, time.Since(start), err == nil, err)
	return err
}

func requireError(label string, err error) error {
	if err == nil {
		return fmt.Errorf("%s unexpectedly succeeded", label)
	}
	return nil
}

func containsServer(xs []contract.Server, id string) bool {
	for _, x := range xs {
		if x.ID == id {
			return true
		}
	}
	return false
}

func containsMember(xs []contract.ServerMember, id string) bool {
	for _, x := range xs {
		if x.IdentityID == id {
			return true
		}
	}
	return false
}

func containsMemberView(xs []contract.Participant, id string) bool {
	for _, x := range xs {
		if x.IdentityID == id {
			return true
		}
	}
	return false
}

func containsGroup(xs []contract.Group, id string) bool {
	for _, x := range xs {
		if x.ID == id {
			return true
		}
	}
	return false
}

func containsContact(xs []contract.Contact, id string) bool { return countContacts(xs, id) > 0 }

func countContacts(xs []contract.Contact, id string) int {
	count := 0
	for _, x := range xs {
		if x.IdentityID == id {
			count++
		}
	}
	return count
}

func containsString(xs []string, value string) bool { return countStrings(xs, value) > 0 }

func countStrings(xs []string, value string) int {
	count := 0
	for _, x := range xs {
		if x == value {
			count++
		}
	}
	return count
}

func containsGroupMember(xs []contract.GroupMember, id string) bool {
	for _, x := range xs {
		if x.IdentityID == id {
			return true
		}
	}
	return false
}

func containsOperation(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func containsNotification(xs []contract.Notification, typ, target string) bool {
	for _, x := range xs {
		if (typ == "" || x.Type == typ) && (target == "" || x.TargetID == target) {
			return true
		}
	}
	return false
}

func (h *Harness) serverSpec(name string, policy contract.JoinPolicy) contract.ServerSpec {
	spec := contract.ServerSpec{
		Name: name, Description: "WalkieBench V2 observable workspace", Purpose: "realtime collaboration benchmark",
		Topics: []string{"simulation", "interoperability"}, Tags: []string{"walkiebench", "v2"}, JoinPolicy: policy,
		Capabilities: []string{"benchmarking", "collaboration"}, Rules: []string{"members-only content", "ordered events"},
		Discoverable: true, ConnectionMethods: []string{"jsonrpc", "browser"},
	}
	h.probe(spec.Description, spec.Purpose)
	for _, value := range append(append(append([]string{}, spec.Topics...), spec.Tags...), spec.Rules...) {
		h.probe(value)
	}
	return spec
}

func (h *Harness) probe(values ...string) {
	for _, value := range values {
		// Very short values are unsuitable probes: a two-character marker can
		// occur by chance in an opaque ciphertext or protocol version string.
		if len([]rune(value)) >= 4 {
			h.t.RegisterPlaintext(value)
		}
	}
}

func (h *Harness) serverLifecycleScenario(ctx context.Context) []error {
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	b, err := h.v2("agent-b")
	if err != nil {
		return []error{err}
	}
	c, err := h.v2("agent-c")
	if err != nil {
		return []error{err}
	}
	public, err := v2Call(a, "CreateServer", func() (contract.Server, error) {
		return a.v2.CreateServer(ctx, h.serverSpec(h.label("research"), contract.JoinPublic))
	})
	if err != nil {
		return []error{err}
	}
	h.serverID = public.ID
	if public.ID == "" || public.OwnerID != a.identityID() || public.JoinPolicy != contract.JoinPublic || public.Description == "" || public.Purpose == "" || len(public.Topics) == 0 || len(public.Tags) == 0 || len(public.Capabilities) == 0 || len(public.Rules) == 0 || !public.Discoverable || len(public.ConnectionMethods) == 0 || public.Version == "" {
		return []error{fmt.Errorf("created Server has incomplete owner or policy state")}
	}
	servers, err := v2Call(a, "ListServers", func() ([]contract.Server, error) {
		return a.v2.ListServers(ctx, contract.ServerQuery{Query: public.Name, Limit: 10})
	})
	if err != nil || !containsServer(servers, public.ID) {
		if err == nil {
			err = fmt.Errorf("ListServers omitted created Server")
		}
		return []error{err}
	}
	servers, err = v2Call(a, "DiscoverServers", func() ([]contract.Server, error) {
		return a.v2.DiscoverServers(ctx, contract.ServerQuery{Tag: "walkiebench", Limit: 10, Response: contract.ResponseOptions{Mode: "compact", Select: []string{"id", "name", "join_policy"}}})
	})
	if err != nil || !containsServer(servers, public.ID) {
		if err == nil {
			err = fmt.Errorf("DiscoverServers omitted discoverable Server")
		}
		return []error{err}
	}
	loaded, err := v2Call(a, "GetServer", func() (contract.Server, error) {
		return a.v2.GetServer(ctx, public.ID, contract.ResponseOptions{Select: []string{"id", "name", "join_policy", "member_count"}})
	})
	if err != nil || loaded.ID != public.ID {
		if err == nil {
			err = fmt.Errorf("GetServer returned the wrong Server")
		}
		return []error{err}
	}
	if err = v2Err(b, "JoinServer", func() error { return b.v2.JoinServer(ctx, public.ID) }); err != nil {
		return []error{err}
	}
	joined, err := a.v2.GetServer(ctx, public.ID, contract.ResponseOptions{})
	if err != nil || joined.MemberCount != 2 {
		if err == nil {
			err = fmt.Errorf("Server member count was %d after join, want 2", joined.MemberCount)
		}
		return []error{err}
	}
	if err = requireError("owner leave", a.v2.LeaveServer(ctx, public.ID)); err != nil {
		return []error{err}
	}
	approval, err := v2Call(a, "CreateServer", func() (contract.Server, error) {
		return a.v2.CreateServer(ctx, h.serverSpec(h.label("approval"), contract.JoinApproval))
	})
	if err != nil {
		return []error{err}
	}
	req, err := v2Call(c, "RequestServerAccess", func() (contract.ServerAccessRequest, error) {
		h.probe("benchmark access request")
		return c.v2.RequestServerAccess(ctx, approval.ID, "benchmark access request")
	})
	if err != nil || req.ID == "" || req.Status != "pending" || req.CreatedAt == "" {
		if err == nil {
			err = fmt.Errorf("access request had no ID")
		}
		return []error{err}
	}
	repeatedRequest, err := v2Call(c, "RequestServerAccessIdempotent", func() (contract.ServerAccessRequest, error) {
		return c.v2.RequestServerAccess(ctx, approval.ID, "benchmark access request")
	})
	if err != nil || repeatedRequest.ID != req.ID {
		if err == nil {
			err = fmt.Errorf("repeated access request created a duplicate")
		}
		return []error{err}
	}
	if _, err = v2Call(c, "ListServerMembersUnauthorized", func() ([]contract.ServerMember, error) {
		return c.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: approval.ID})
	}); err == nil {
		return []error{fmt.Errorf("approval-required Server exposed its private member directory before approval")}
	}
	requests, err := v2Call(a, "ListServerRequests", func() ([]contract.ServerAccessRequest, error) {
		return a.v2.ListServerRequests(ctx, approval.ID, contract.ResponseOptions{Mode: "compact"})
	})
	if err != nil || len(requests) != 1 || requests[0].ID != req.ID || requests[0].Status != "pending" {
		if err == nil {
			err = fmt.Errorf("owner could not see pending access request")
		}
		return []error{err}
	}
	if err = v2Err(a, "ApproveServerRequest", func() error { return a.v2.ApproveServerRequest(ctx, req.ID) }); err != nil {
		return []error{err}
	}
	requests, err = a.v2.ListServerRequests(ctx, approval.ID, contract.ResponseOptions{})
	if err != nil || len(requests) != 1 || requests[0].Status != "approved" {
		if err == nil {
			err = fmt.Errorf("approved request lifecycle was not visible")
		}
		return []error{err}
	}
	if err = v2Err(c, "JoinApprovedServer", func() error { return c.v2.JoinServer(ctx, approval.ID) }); err != nil {
		return []error{err}
	}
	h.t.Metric("job_c_approval_workflow", 1)
	inviteOnly, err := v2Call(a, "CreateServerInviteOnly", func() (contract.Server, error) {
		return a.v2.CreateServer(ctx, h.serverSpec(h.label("invite"), contract.JoinInvite))
	})
	if err != nil {
		return []error{err}
	}
	if err = requireError("invite-only self join", func() error { return c.v2.JoinServer(ctx, inviteOnly.ID) }()); err != nil {
		return []error{err}
	}
	inv, err := v2Call(a, "InviteToServer", func() (contract.ServerInvite, error) {
		return a.v2.InviteToServer(ctx, inviteOnly.ID, c.identityID())
	})
	if err != nil || inv.ID == "" || inv.Status != "pending" || inv.InvitedBy != a.identityID() || inv.CreatedAt == "" || inv.ExpiresAt == "" {
		if err == nil {
			err = fmt.Errorf("server invitation had no ID")
		}
		return []error{err}
	}
	invites, err := v2Call(c, "ListServerInvites", func() ([]contract.ServerInvite, error) {
		return c.v2.ListServerInvites(ctx, inviteOnly.ID, contract.ResponseOptions{})
	})
	if err != nil || len(invites) == 0 {
		if err == nil {
			err = fmt.Errorf("invitee could not retrieve Server invitation")
		}
		return []error{err}
	}
	if err = v2Err(c, "AcceptServerInvite", func() error { return c.v2.AcceptServerInvite(ctx, inv.ID) }); err != nil {
		return []error{err}
	}
	if err = v2Err(c, "AcceptServerInviteIdempotent", func() error { return c.v2.AcceptServerInvite(ctx, inv.ID) }); err != nil {
		return []error{fmt.Errorf("repeated acceptance was not idempotent: %w", err)}
	}
	inviteMembers, err := c.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: inviteOnly.ID})
	if err != nil || !containsMember(inviteMembers, c.identityID()) {
		if err == nil {
			err = fmt.Errorf("accepted invite did not create membership")
		}
		return []error{err}
	}
	if err = v2Err(c, "LeaveServer", func() error { return c.v2.LeaveServer(ctx, inviteOnly.ID) }); err != nil {
		return []error{err}
	}
	if _, err = c.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: inviteOnly.ID}); err == nil {
		return []error{fmt.Errorf("left member retained Server access")}
	}
	closed, err := v2Call(a, "CreateClosedServer", func() (contract.Server, error) {
		return a.v2.CreateServer(ctx, h.serverSpec(h.label("closed"), contract.JoinClosed))
	})
	if err != nil {
		return []error{err}
	}
	if err = requireError("closed self join", func() error { return c.v2.JoinServer(ctx, closed.ID) }()); err != nil {
		return []error{err}
	}
	if _, err = v2Call(c, "ListClosedServerMembers", func() ([]contract.ServerMember, error) {
		return c.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: closed.ID})
	}); err == nil {
		return []error{fmt.Errorf("cross-Server member enumeration was accepted")}
	}
	discoveredClosed, err := c.v2.DiscoverServers(ctx, contract.ServerQuery{Query: closed.Name, Limit: 10})
	if err != nil || !containsServer(discoveredClosed, closed.ID) {
		if err == nil {
			err = fmt.Errorf("discoverable closed Server hid its public onboarding metadata")
		}
		return []error{err}
	}
	if err = v2Err(a, "RemoveServerMember", func() error { return a.v2.RemoveServerMember(ctx, approval.ID, c.identityID()) }); err != nil {
		return []error{err}
	}
	if _, err = c.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: approval.ID}); err == nil {
		return []error{fmt.Errorf("removed Server member retained access")}
	}
	publicPolicy := contract.JoinPublic
	updatedDescription := "updated benchmark description"
	updatedPurpose := "updated benchmark purpose"
	updatedName := h.label("updated-server")
	discoverable := false
	updated, err := v2Call(a, "UpdateServer", func() (contract.Server, error) {
		return a.v2.UpdateServer(ctx, approval.ID, contract.ServerPatch{Name: &updatedName, Description: &updatedDescription, Purpose: &updatedPurpose, JoinPolicy: &publicPolicy, Discoverable: &discoverable})
	})
	if err != nil || updated.JoinPolicy != contract.JoinPublic || updated.Name != updatedName || updated.Description != updatedDescription || updated.Purpose != updatedPurpose || updated.Discoverable {
		if err == nil {
			err = fmt.Errorf("Server join-policy transition was not visible")
		}
		return []error{err}
	}
	reloaded, err := a.v2.GetServer(ctx, approval.ID, contract.ResponseOptions{})
	if err != nil || reloaded.Name != updatedName || reloaded.Description != updatedDescription || reloaded.Purpose != updatedPurpose || reloaded.JoinPolicy != contract.JoinPublic || reloaded.Discoverable {
		if err == nil {
			err = fmt.Errorf("updated Server metadata was not durable on read-back")
		}
		return []error{err}
	}
	invalidPolicy := contract.JoinPolicy("invalid")
	if _, err = a.v2.UpdateServer(ctx, approval.ID, contract.ServerPatch{JoinPolicy: &invalidPolicy}); err == nil {
		return []error{fmt.Errorf("invalid Server join policy was accepted")}
	}
	rejectServer, err := v2Call(a, "CreateRejectServer", func() (contract.Server, error) {
		return a.v2.CreateServer(ctx, h.serverSpec(h.label("reject"), contract.JoinApproval))
	})
	if err != nil {
		return []error{err}
	}
	rejectReq, err := v2Call(c, "RequestRejectServerAccess", func() (contract.ServerAccessRequest, error) {
		return c.v2.RequestServerAccess(ctx, rejectServer.ID, "expected rejection")
	})
	if err != nil {
		return []error{err}
	}
	if err = v2Err(a, "RejectServerRequest", func() error { return a.v2.RejectServerRequest(ctx, rejectReq.ID) }); err != nil {
		return []error{err}
	}
	if err = requireError("rejected Server join", func() error { return c.v2.JoinServer(ctx, rejectServer.ID) }()); err != nil {
		return []error{err}
	}
	rejected, err := a.v2.ListServerRequests(ctx, rejectServer.ID, contract.ResponseOptions{})
	if err != nil || len(rejected) != 1 || rejected[0].Status != "rejected" {
		if err == nil {
			err = fmt.Errorf("rejected request lifecycle was not visible")
		}
		return []error{err}
	}
	return nil
}

func (h *Harness) serverPermissionsScenario(ctx context.Context) []error {
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	b, err := h.v2("agent-b")
	if err != nil {
		return []error{err}
	}
	c, err := h.v2("agent-c")
	if err != nil {
		return []error{err}
	}
	human, err := h.v2("human")
	if err != nil {
		return []error{err}
	}
	if _, err = human.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID}); err == nil {
		return []error{fmt.Errorf("non-member enumerated the Server directory")}
	}
	if err = c.v2.JoinServer(ctx, h.serverID); err != nil {
		return []error{err}
	}
	if err = human.v2.JoinServer(ctx, h.serverID); err != nil {
		return []error{err}
	}
	members, err := v2Call(a, "ListServerMembers", func() ([]contract.ServerMember, error) {
		return a.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID, Capability: "collaboration", Limit: 20, Response: contract.ResponseOptions{Mode: "compact"}})
	})
	if err != nil || !containsMember(members, b.identityID()) {
		if err == nil {
			err = fmt.Errorf("authorized member discovery omitted agent-b")
		}
		return []error{err}
	}
	var discoveredAgent bool
	for _, candidate := range members {
		if candidate.IdentityID == b.identityID() {
			discoveredAgent = candidate.Kind == "agent" && candidate.DisplayName != "" && candidate.Harness == "benchmark" && len(candidate.Capabilities) > 0 && candidate.CurrentWork != "" && len(candidate.CollaborationTopics) > 0 && !candidate.LastSeen.IsZero()
		}
	}
	if !discoveredAgent {
		return []error{fmt.Errorf("member directory omitted agent type, profile, capability, work, topic, or activity metadata")}
	}
	humans, err := a.v2.FindServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID, Name: "human", Limit: 10})
	if err != nil || len(humans) != 1 || humans[0].IdentityID != human.identityID() || humans[0].Kind != "human" {
		if err == nil {
			err = fmt.Errorf("human type and name filtering were not preserved")
		}
		return []error{err}
	}
	filtered, err := v2Call(a, "FindServerMembers", func() ([]contract.ServerMember, error) {
		return a.v2.FindServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID, Harness: "benchmark", Topic: "realtime", Online: boolPtr(true), Limit: 10})
	})
	if err != nil || len(filtered) == 0 {
		if err == nil {
			err = fmt.Errorf("filtered member discovery returned no authorized participants")
		}
		return []error{err}
	}
	member, err := v2Call(a, "GetServerMember", func() (contract.ServerMember, error) {
		return a.v2.GetServerMember(ctx, h.serverID, b.identityID(), contract.ResponseOptions{Select: []string{"id", "display_name", "role", "capabilities"}})
	})
	if err != nil || member.IdentityID != b.identityID() {
		if err == nil {
			err = fmt.Errorf("GetServerMember returned wrong participant")
		}
		return []error{err}
	}
	if err = v2Err(a, "SetServerRoleAdmin", func() error { return a.v2.SetServerRole(ctx, h.serverID, b.identityID(), contract.RoleAdmin) }); err != nil {
		return []error{err}
	}
	if err = v2Err(a, "SetServerRoleModerator", func() error { return a.v2.SetServerRole(ctx, h.serverID, c.identityID(), contract.RoleModerator) }); err != nil {
		return []error{err}
	}
	if err = a.v2.SetServerRole(ctx, h.serverID, human.identityID(), contract.MemberRole("superuser")); err == nil {
		return []error{fmt.Errorf("unknown Server role was accepted")}
	}
	roles, err := v2Call(a, "ListServerRoles", func() ([]contract.ServerRole, error) { return a.v2.ListServerRoles(ctx, h.serverID) })
	if err != nil || len(roles) < 2 {
		if err == nil {
			err = fmt.Errorf("role listing omitted configured roles")
		}
		return []error{err}
	}
	var adminRole, moderatorRole bool
	for _, role := range roles {
		if role.Participant == b.identityID() && role.Role == contract.RoleAdmin {
			adminRole = true
		}
		if role.Participant == c.identityID() && role.Role == contract.RoleModerator {
			moderatorRole = true
		}
	}
	if !adminRole || !moderatorRole {
		return []error{fmt.Errorf("role transitions were not visible in role listing")}
	}
	if err = requireError("member permission escalation", v2Err(c, "UnauthorizedPermissionChange", func() error {
		return c.v2.UpdateServerPermissions(ctx, h.serverID, contract.PermissionChange{Role: contract.RoleMember, Permission: "manage_roles", Allowed: true})
	})); err != nil {
		return []error{err}
	}
	if err = v2Err(a, "UpdateServerPermissions", func() error {
		return a.v2.UpdateServerPermissions(ctx, h.serverID, contract.PermissionChange{Role: contract.RoleModerator, Permission: "moderate_posts", Allowed: true})
	}); err != nil {
		return []error{err}
	}
	audit, err := v2Call(a, "GetServerAudit", func() ([]contract.ActivityEvent, error) {
		return a.v2.GetServerAudit(ctx, h.serverID, contract.ResponseOptions{Limit: 50})
	})
	if err != nil || len(audit) == 0 {
		if err == nil {
			err = fmt.Errorf("authorized audit view was empty after administration")
		}
		return []error{err}
	}
	return nil
}

func (h *Harness) serverGroupsScenario(ctx context.Context) []error {
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	b, err := h.v2("agent-b")
	if err != nil {
		return []error{err}
	}
	c, err := h.v2("agent-c")
	if err != nil {
		return []error{err}
	}
	public, err := v2Call(a, "CreateServerGroup", func() (contract.Group, error) {
		return a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: h.serverID, Name: h.label("public-group"), JoinPolicy: contract.JoinPublic})
	})
	if err != nil {
		return []error{err}
	}
	h.groupIDV2 = public.ID
	if public.ServerID != h.serverID || public.OwnerID != a.identityID() || public.JoinPolicy != contract.JoinPublic || public.Private {
		return []error{fmt.Errorf("created group omitted Server, owner, or policy metadata")}
	}
	if err = v2Err(b, "JoinGroup", func() error { return b.v2.JoinGroupV2(ctx, public.ID) }); err != nil {
		return []error{err}
	}
	cleanup, err := v2Call(a, "CreateCleanupGroup", func() (contract.Group, error) {
		return a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: h.serverID, Name: h.label("cleanup-group"), JoinPolicy: contract.JoinPublic})
	})
	if err != nil {
		return []error{err}
	}
	if err = v2Err(b, "JoinCleanupGroup", func() error { return b.v2.JoinGroupV2(ctx, cleanup.ID) }); err != nil {
		return []error{err}
	}
	if err = v2Err(c, "JoinCleanupGroupAsSecondMember", func() error { return c.v2.JoinGroupV2(ctx, cleanup.ID) }); err != nil {
		return []error{err}
	}
	if err = v2Err(c, "LeaveGroup", func() error { return c.v2.LeaveGroupV2(ctx, cleanup.ID) }); err != nil {
		return []error{err}
	}
	if err = v2Err(a, "RemoveGroupMember", func() error { return a.v2.RemoveGroupMember(ctx, cleanup.ID, b.identityID()) }); err != nil {
		return []error{err}
	}
	groups, err := v2Call(b, "DiscoverGroups", func() ([]contract.Group, error) {
		return b.v2.DiscoverGroups(ctx, contract.GroupQuery{ServerID: h.serverID, Query: public.Name, Limit: 10})
	})
	if err != nil || len(groups) == 0 {
		if err == nil {
			err = fmt.Errorf("DiscoverGroups omitted public group")
		}
		return []error{err}
	}
	listed, err := v2Call(a, "ListServerGroups", func() ([]contract.Group, error) {
		return a.v2.ListServerGroups(ctx, contract.GroupQuery{ServerID: h.serverID, Limit: 10, Response: contract.ResponseOptions{Mode: "compact"}})
	})
	if err != nil || len(listed) == 0 {
		if err == nil {
			err = fmt.Errorf("ListGroups omitted group")
		}
		return []error{err}
	}
	approval, err := v2Call(a, "CreateApprovalGroup", func() (contract.Group, error) {
		return a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: h.serverID, Name: h.label("approval-group"), JoinPolicy: contract.JoinApproval})
	})
	if err != nil {
		return []error{err}
	}
	request, err := v2Call(c, "RequestGroupAccess", func() (contract.GroupAccessRequest, error) {
		h.probe("group access")
		return c.v2.RequestGroupAccess(ctx, approval.ID, "group access")
	})
	if err != nil {
		return []error{err}
	}
	repeatedRequest, err := c.v2.RequestGroupAccess(ctx, approval.ID, "group access")
	if err != nil || repeatedRequest.ID != request.ID {
		if err == nil {
			err = fmt.Errorf("repeated group access request created a duplicate")
		}
		return []error{err}
	}
	requests, err := a.v2.ListGroupRequests(ctx, approval.ID, contract.ResponseOptions{Mode: "compact"})
	if err != nil || len(requests) != 1 || requests[0].ID != request.ID || requests[0].Status != "pending" {
		if err == nil {
			err = fmt.Errorf("group owner could not inspect the pending request")
		}
		return []error{err}
	}
	if _, err = c.v2.ListGroupRequests(ctx, approval.ID, contract.ResponseOptions{}); err == nil {
		return []error{fmt.Errorf("ordinary member enumerated group access requests")}
	}
	if err = v2Err(a, "ApproveGroupRequest", func() error { return a.v2.ApproveGroupRequest(ctx, request.ID) }); err != nil {
		return []error{err}
	}
	if err = v2Err(c, "JoinApprovedGroup", func() error { return c.v2.JoinGroupV2(ctx, approval.ID) }); err != nil {
		return []error{err}
	}
	rejectedGroup, err := v2Call(a, "CreateRejectedGroup", func() (contract.Group, error) {
		return a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: h.serverID, Name: h.label("rejected-group"), JoinPolicy: contract.JoinApproval})
	})
	if err != nil {
		return []error{err}
	}
	rejectedRequest, err := v2Call(c, "RequestRejectedGroupAccess", func() (contract.GroupAccessRequest, error) {
		h.probe("rejected group access")
		return c.v2.RequestGroupAccess(ctx, rejectedGroup.ID, "rejected group access")
	})
	if err != nil {
		return []error{err}
	}
	if err = v2Err(a, "RejectGroupRequest", func() error { return a.v2.RejectGroupRequest(ctx, rejectedRequest.ID) }); err != nil {
		return []error{err}
	}
	if err = requireError("rejected group join", v2Err(c, "JoinRejectedGroup", func() error { return c.v2.JoinGroupV2(ctx, rejectedGroup.ID) })); err != nil {
		return []error{err}
	}
	invite, err := v2Call(a, "CreateInviteGroup", func() (contract.Group, error) {
		return a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: h.serverID, Name: h.label("invite-group"), JoinPolicy: contract.JoinInvite})
	})
	if err != nil {
		return []error{err}
	}
	if err = requireError("invite-only group self join", v2Err(c, "UnauthorizedGroupJoin", func() error { return c.v2.JoinGroupV2(ctx, invite.ID) })); err != nil {
		return []error{err}
	}
	gi, err := v2Call(a, "InviteToGroup", func() (contract.GroupInvite, error) { return a.v2.InviteToGroup(ctx, invite.ID, c.identityID()) })
	if err != nil {
		return []error{err}
	}
	repeatedInvite, err := a.v2.InviteToGroup(ctx, invite.ID, c.identityID())
	if err != nil || repeatedInvite.ID != gi.ID {
		if err == nil {
			err = fmt.Errorf("repeated group invitation created a duplicate")
		}
		return []error{err}
	}
	invites, err := c.v2.ListGroupInvites(ctx, invite.ID, contract.ResponseOptions{})
	if err != nil || len(invites) != 1 || invites[0].ID != gi.ID || invites[0].InvitedBy != a.identityID() {
		if err == nil {
			err = fmt.Errorf("group invitee could not inspect its invitation")
		}
		return []error{err}
	}
	if err = v2Err(c, "AcceptGroupInvite", func() error { return c.v2.AcceptGroupInvite(ctx, gi.ID) }); err != nil {
		return []error{err}
	}
	private, err := v2Call(a, "CreatePrivateGroup", func() (contract.Group, error) {
		return a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: h.serverID, Name: h.label("private-group"), JoinPolicy: contract.JoinClosed, Private: true})
	})
	if err != nil {
		return []error{err}
	}
	if err = requireError("private group unauthorized join", v2Err(c, "UnauthorizedPrivateGroupJoin", func() error { return c.v2.JoinGroupV2(ctx, private.ID) })); err != nil {
		return []error{err}
	}
	privateDiscovery, err := c.v2.DiscoverGroups(ctx, contract.GroupQuery{ServerID: h.serverID, Query: private.Name, Limit: 10})
	if err != nil || len(privateDiscovery) != 0 {
		if err == nil {
			err = fmt.Errorf("private group leaked through discovery")
		}
		return []error{err}
	}
	if err = v2Err(a, "UpdateGroup", func() error {
		name := h.label("renamed-group")
		_, e := a.v2.UpdateGroupV2(ctx, public.ID, contract.GroupPatch{Name: &name})
		return e
	}); err != nil {
		return []error{err}
	}
	members, err := v2Call(a, "ListGroupMembers", func() ([]contract.GroupMember, error) {
		return a.v2.ListGroupMembers(ctx, public.ID, contract.ResponseOptions{Limit: 10})
	})
	if err != nil || !containsGroupMember(members, b.identityID()) {
		if err == nil {
			err = fmt.Errorf("group member listing omitted joined member")
		}
		return []error{err}
	}
	var joinedRole contract.GroupRole
	for _, member := range members {
		if member.IdentityID == b.identityID() {
			joinedRole = member.GroupRole
		}
	}
	if joinedRole != contract.GroupRoleMember {
		return []error{fmt.Errorf("joined group member has role %q", joinedRole)}
	}
	if err = a.v2.SetGroupRole(ctx, public.ID, b.identityID(), contract.GroupRoleAdmin); err != nil {
		return []error{err}
	}
	if err = c.v2.SetGroupRole(ctx, public.ID, c.identityID(), contract.GroupRoleAdmin); err == nil {
		return []error{fmt.Errorf("non-member manipulated a group role")}
	}
	if err = a.v2.UpdateGroupPermissions(ctx, public.ID, contract.GroupPermissionChange{Role: contract.GroupRoleMember, Permission: contract.GroupPermissionSend, Allowed: true}); err != nil {
		return []error{err}
	}
	jobBMessage := "job-b-group-context-" + h.label("message")
	if _, err = b.SendGroupMessage(ctx, public.ID, jobBMessage); err != nil {
		return []error{err}
	}
	groupHistory, err := a.GetGroupHistory(ctx, public.ID)
	if err != nil || !hasContent(groupHistory, jobBMessage) {
		if err == nil {
			err = fmt.Errorf("group collaboration job could not retrieve recent context")
		}
		return []error{err}
	}
	h.t.Metric("job_b_join_collaboration", 1)
	if err = v2Err(a, "DeleteGroup", func() error { return a.v2.DeleteGroupV2(ctx, private.ID) }); err != nil {
		return []error{err}
	}
	return nil
}

func (h *Harness) serverForumsScenario(ctx context.Context) []error {
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	b, err := h.v2("agent-b")
	if err != nil {
		return []error{err}
	}
	c, err := h.v2("agent-c")
	if err != nil {
		return []error{err}
	}
	post, err := v2Call(a, "CreateServerPost", func() (contract.PostView, error) {
		h.probe("v2 forum content")
		title := h.label("forum-post")
		h.probe(title)
		return a.v2.CreateServerPost(ctx, contract.PostSpec{ServerID: h.serverID, Title: title, Content: "v2 forum content", Visibility: contract.VisibilityServer, Mentions: []string{b.identityID()}})
	})
	if err != nil {
		return []error{err}
	}
	h.postIDV2 = post.ID
	posts, err := v2Call(b, "DiscoverPosts", func() ([]contract.PostView, error) {
		return b.v2.DiscoverPosts(ctx, contract.PostQuery{ServerID: h.serverID, Query: post.Title, Limit: 10})
	})
	if err != nil || len(posts) == 0 {
		if err == nil {
			err = fmt.Errorf("DiscoverPosts omitted server-wide post")
		}
		return []error{err}
	}
	searched, err := v2Call(b, "SearchPosts", func() ([]contract.PostView, error) {
		return b.v2.SearchPosts(ctx, contract.PostQuery{ServerID: h.serverID, Query: "v2 forum", Limit: 10, Response: contract.ResponseOptions{Mode: "compact"}})
	})
	if err != nil || len(searched) == 0 {
		if err == nil {
			err = fmt.Errorf("SearchPosts omitted matching content")
		}
		return []error{err}
	}
	if err = v2Err(b, "FollowThread", func() error { return b.FollowThread(ctx, post.ID) }); err != nil {
		return []error{err}
	}
	root, err := v2Call(b, "CommentServer", func() (contract.CommentNode, error) {
		h.probe("v2 root reply")
		return b.v2.CommentServer(ctx, post.ID, "v2 root reply", []string{c.identityID()})
	})
	if err != nil {
		return []error{err}
	}
	child, err := v2Call(c, "NestedCommentServer", func() (contract.CommentNode, error) {
		h.probe("v2 nested reply")
		return c.v2.CommentServer(ctx, root.ID, "v2 nested reply", nil)
	})
	if err != nil || child.ParentID != root.ID {
		if err == nil {
			err = fmt.Errorf("nested V2 comment has incorrect parent")
		}
		return []error{err}
	}
	if err = v2Err(a, "React", func() error { return a.React(ctx, post.ID, "like") }); err != nil {
		return []error{err}
	}
	thread, err := v2Call(b, "GetServerThread", func() (contract.Thread, error) {
		return b.v2.GetServerThread(ctx, post.ID, contract.ResponseOptions{Mode: "compact"})
	})
	if err != nil || countComments(thread.Comments) < 2 {
		if err == nil {
			err = fmt.Errorf("V2 thread omitted nested comments")
		}
		return []error{err}
	}
	notifications, err := v2Call(b, "ListNotifications", func() ([]contract.Notification, error) {
		return b.v2.ListNotifications(ctx, contract.NotificationQuery{UnreadOnly: true, Limit: 50, Response: contract.ResponseOptions{Mode: "compact"}})
	})
	if err != nil || (!containsNotification(notifications, "mention", post.ID) && !containsNotification(notifications, "reply", post.ID)) {
		if err == nil {
			err = fmt.Errorf("followed-thread or mention notification was not delivered")
		}
		return []error{err}
	}
	if len(notifications) > 0 {
		ids := make([]string, 0, len(notifications))
		for _, n := range notifications {
			ids = append(ids, n.ID)
		}
		if err = v2Err(b, "MarkNotificationsRead", func() error { return b.v2.MarkNotificationsRead(ctx, ids) }); err != nil {
			return []error{err}
		}
	}
	if err = v2Err(a, "SharePost", func() error { return a.v2.SharePost(ctx, post.ID, h.serverID, h.groupIDV2) }); err != nil {
		return []error{err}
	}
	groupPost, err := v2Call(a, "CreateGroupPost", func() (contract.PostView, error) {
		h.probe("group-private content")
		title := h.label("group-post")
		h.probe(title)
		return a.v2.CreateServerPost(ctx, contract.PostSpec{ServerID: h.serverID, GroupID: h.groupIDV2, Title: title, Content: "group-private content", Visibility: contract.VisibilityGroup})
	})
	if err != nil {
		return []error{err}
	}
	if _, err = v2Call(c, "UnauthorizedGroupPostThread", func() (contract.Thread, error) {
		return c.v2.GetServerThread(ctx, groupPost.ID, contract.ResponseOptions{})
	}); err == nil {
		return []error{fmt.Errorf("non-member retrieved group-only post")}
	}
	if err = v2Err(b, "UnfollowThread", func() error { return b.UnfollowThread(ctx, post.ID) }); err != nil {
		return []error{err}
	}
	h.t.Metric("job_e_forum_collaboration", 1)
	editedTitle := h.label("edited-post")
	h.probe(editedTitle, "edited forum content")
	edited, err := v2Call(a, "EditServerPost", func() (contract.PostView, error) {
		return a.v2.EditServerPost(ctx, post.ID, contract.PostPatch{Title: &editedTitle, Content: stringPtr("edited forum content")})
	})
	if err != nil || edited.ID != post.ID {
		if err == nil {
			err = fmt.Errorf("EditPost returned the wrong post")
		}
		return []error{err}
	}
	return nil
}

func (h *Harness) declarativeScenario(ctx context.Context) []error {
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	b, err := h.v2("agent-b")
	if err != nil {
		return []error{err}
	}
	manifestServer, err := b.v2.CreateServer(ctx, h.serverSpec(h.label("manifest-server"), contract.JoinPublic))
	if err != nil {
		return []error{err}
	}
	manifestGroup, err := b.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: manifestServer.ID, Name: h.label("manifest-group"), JoinPolicy: contract.JoinPublic})
	if err != nil {
		return []error{err}
	}
	manifestPost, err := b.v2.CreateServerPost(ctx, contract.PostSpec{ServerID: manifestServer.ID, Title: h.label("manifest-post"), Content: "manifest follow target", Visibility: contract.VisibilityServer})
	if err != nil {
		return []error{err}
	}
	before, err := b.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: manifestServer.ID})
	if err != nil {
		return []error{err}
	}
	contactsBefore, err := a.ListContacts(ctx)
	if err != nil {
		return []error{err}
	}
	manifest := contract.Manifest{APIVersion: "harnesstalkie/v2", Kind: "Session", Server: manifestServer.ID, Identity: contract.ManifestIdentity{Name: "agent-a", Profile: &contract.Profile{DisplayName: "agent-a", Kind: "agent", Harness: "benchmark", Capabilities: []string{"collaboration"}}}, Membership: contract.ManifestMembership{Join: "if-allowed", RequestIfRequired: true, AcceptInvitation: true}, Discover: contract.ManifestDiscovery{Capabilities: []string{"collaboration"}, Limit: 5}, Groups: contract.ManifestGroups{Discover: true, JoinPublic: true, Limit: 5}, Contacts: []string{b.identityID()}, Follows: []string{manifestPost.ID}, Sync: contract.ManifestSync{Inbox: true, Mentions: true, Since: "last"}, Presence: contract.ManifestPresence{Online: true}, Response: contract.ResponseOptions{Mode: "compact", Select: []string{"id", "name", "status", "capabilities"}}}
	h.probe("collaboration")
	if err = contract.ValidateManifest(manifest); err != nil {
		return []error{err}
	}
	h.t.BeginAgentJob("one-shot-declarative-bootstrap")
	first, err := v2Call(a, "ApplyManifest", func() (contract.ManifestResult, error) { return a.v2.ApplyManifest(ctx, manifest) })
	if err != nil {
		h.t.EndAgentJob(false)
		return []error{err}
	}
	second, err := v2Call(a, "ApplyManifestIdempotent", func() (contract.ManifestResult, error) { return a.v2.ApplyManifest(ctx, manifest) })
	if err != nil || first.Server.ID != second.Server.ID || first.Identity.ID != second.Identity.ID {
		h.t.EndAgentJob(false)
		if err == nil {
			err = fmt.Errorf("repeated manifest changed desired identity or Server")
		}
		return []error{err}
	}
	h.t.EndAgentJob(true)
	if first.Membership != "joined" || second.Membership != "already-member" || first.Server.ID != manifestServer.ID || !containsMemberView(first.Participants, b.identityID()) || !containsGroup(first.Groups, manifestGroup.ID) || !containsContact(first.Contacts, b.identityID()) || !containsString(first.Follows, manifestPost.ID) || first.Cursor == 0 {
		return []error{fmt.Errorf("manifest did not realize membership, discovery, group, contact, follow, presence, and sync outcomes")}
	}
	after, err := v2Call(a, "ListMembersAfterManifest", func() ([]contract.ServerMember, error) {
		return a.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: manifestServer.ID})
	})
	if err != nil || len(after) != len(before)+1 {
		if err == nil {
			err = fmt.Errorf("manifest application duplicated membership")
		}
		return []error{err}
	}
	groupMembers, err := a.v2.ListGroupMembers(ctx, manifestGroup.ID, contract.ResponseOptions{})
	if err != nil || !containsGroupMember(groupMembers, a.identityID()) {
		if err == nil {
			err = fmt.Errorf("manifest did not join an eligible public group")
		}
		return []error{err}
	}
	contactsAfter, err := a.ListContacts(ctx)
	if err != nil || countContacts(contactsAfter, b.identityID()) != 1 || len(contactsAfter) < len(contactsBefore) {
		if err == nil {
			err = fmt.Errorf("manifest contact state was duplicated or lost")
		}
		return []error{err}
	}
	thread, err := a.GetThread(ctx, manifestPost.ID)
	if err != nil || countStrings(thread.Followers, a.identityID()) != 1 {
		if err == nil {
			err = fmt.Errorf("manifest follow state was duplicated")
		}
		return []error{err}
	}
	h.t.Metric("job_f_declarative_bootstrap", 1)
	batch := contract.BatchRequest{Mode: "ordered", Operations: []contract.BatchOperation{
		{ID: "group", Operation: "CreateGroup", Params: map[string]any{"server_id": manifestServer.ID, "name": h.label("batch-group"), "join_policy": "public"}},
		{ID: "members", Operation: "ListGroupMembers", DependsOn: []string{"group"}, Params: map[string]any{"group_id": "$ref:group.id", "limit": 5}},
	}}
	if err = contract.ValidateBatch(batch); err != nil {
		return []error{err}
	}
	primitiveStart := time.Now()
	primitiveGroup, err := v2Call(a, "PrimitiveWorkflowCreateGroup", func() (contract.Group, error) {
		return a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: manifestServer.ID, Name: h.label("primitive-group"), JoinPolicy: contract.JoinPublic})
	})
	if err != nil {
		return []error{err}
	}
	if _, err = v2Call(a, "PrimitiveWorkflowListMembers", func() ([]contract.GroupMember, error) {
		return a.v2.ListGroupMembers(ctx, primitiveGroup.ID, contract.ResponseOptions{Limit: 5})
	}); err != nil {
		return []error{err}
	}
	h.t.Metric("primitive_workflow_round_trips", 2)
	h.t.Metric("primitive_workflow_latency_ms", float64(time.Since(primitiveStart).Microseconds())/1000)
	h.t.Metric("manifest_application_operations", 2)
	start := time.Now()
	response, err := v2Call(a, "Batch", func() (contract.BatchResponse, error) { return a.v2.Batch(ctx, batch) })
	h.t.Metric("batch_round_trips", float64(response.RoundTrips))
	h.t.Metric("batch_latency_ms", float64(time.Since(start).Microseconds())/1000)
	if err != nil || response.RoundTrips != 1 || len(response.Results) != 2 || response.Results[0].ID != "group" || response.Results[1].ID != "members" || !response.Results[0].Success || !response.Results[1].Success {
		if err == nil {
			err = fmt.Errorf("batch response did not preserve dependency order")
		}
		return []error{err}
	}
	failedBatch, batchErr := v2Call(a, "BatchPartialFailure", func() (contract.BatchResponse, error) {
		return a.v2.Batch(ctx, contract.BatchRequest{Mode: "ordered", Operations: []contract.BatchOperation{{ID: "missing", Operation: "GetServer", Params: map[string]any{"server_id": "server-does-not-exist"}}, {ID: "dependent", Operation: "ListServerMembers", DependsOn: []string{"missing"}, Params: map[string]any{"server_id": "$ref:missing.id"}}, {ID: "independent", Operation: "ListServers", Params: map[string]any{"limit": 1}}}})
	})
	if batchErr == nil {
		partialFailure, dependencySkipped, independentRan := false, false, false
		for _, result := range failedBatch.Results {
			if result.ID == "missing" && !result.Success {
				partialFailure = true
			}
			if result.ID == "dependent" && !result.Success && result.Error != nil && result.Error.Code == "dependency_failed" {
				dependencySkipped = true
			}
			if result.ID == "independent" && result.Success {
				independentRan = true
			}
		}
		if !partialFailure || !dependencySkipped || !independentRan {
			return []error{fmt.Errorf("batch did not isolate partial failure and skip its dependent")}
		}
	}
	malformed := contract.BatchRequest{Operations: []contract.BatchOperation{{ID: "duplicate", Operation: "ListServers"}, {ID: "duplicate", Operation: "ListServers"}}}
	if _, err = a.v2.Batch(ctx, malformed); err == nil {
		return []error{fmt.Errorf("server accepted a malformed batch with duplicate operation IDs")}
	}
	return nil
}

func (h *Harness) deltaScenario(ctx context.Context) []error {
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	initial, err := v2Call(a, "SyncInitial", func() (contract.Delta, error) {
		return a.v2.Sync(ctx, contract.SyncRequest{ServerID: h.serverID, Include: []string{"messages", "comments", "notifications", "members"}, Response: contract.ResponseOptions{Mode: "compact"}})
	})
	if err != nil {
		return []error{err}
	}
	message := "delta-message-" + h.label("probe")
	if _, err = a.SendGroupMessage(ctx, h.groupIDV2, message); err != nil {
		return []error{err}
	}
	if _, err = v2Call(a, "DeltaComment", func() (contract.CommentNode, error) { return a.v2.CommentServer(ctx, h.postIDV2, "delta-comment", nil) }); err != nil {
		return []error{err}
	}
	delta, err := v2Call(a, "SyncDelta", func() (contract.Delta, error) {
		return a.v2.Sync(ctx, contract.SyncRequest{ServerID: h.serverID, AfterCursor: initial.Cursor, Limit: 100, Include: []string{"messages", "comments", "notifications"}, Response: contract.ResponseOptions{Mode: "compact"}})
	})
	if err != nil {
		return []error{err}
	}
	if delta.Cursor <= initial.Cursor || len(delta.Events)+len(delta.Messages)+len(delta.Notifications) == 0 {
		return []error{fmt.Errorf("delta sync did not return new authorized activity")}
	}
	if delta.Cursor <= initial.Cursor {
		return []error{fmt.Errorf("delta cursor did not advance")}
	}
	h.t.Metric("job_d_async_resume", 1)
	h.t.Metric("delta_events_returned", float64(len(delta.Events)+len(delta.Messages)+len(delta.Notifications)))
	full, err := v2Call(a, "GetServerFull", func() (contract.Server, error) { return a.v2.GetServer(ctx, h.serverID, contract.ResponseOptions{}) })
	if err != nil {
		return []error{err}
	}
	compact, err := v2Call(a, "GetServerCompact", func() (contract.Server, error) {
		return a.v2.GetServer(ctx, h.serverID, contract.ResponseOptions{Mode: "compact", Select: []string{"id", "name", "join_policy"}})
	})
	if err != nil {
		return []error{err}
	}
	fullBytes, _ := json.Marshal(full)
	compactBytes, _ := json.Marshal(compact)
	if len(compactBytes) >= len(fullBytes) {
		return []error{fmt.Errorf("compact response was not smaller than full response")}
	}
	h.t.Metric("response_shaping_reduction_ratio", 1-float64(len(compactBytes))/float64(len(fullBytes)))
	page, err := v2Call(a, "ListMembersLimited", func() ([]contract.ServerMember, error) {
		return a.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID, Limit: 1, Cursor: 0})
	})
	if err != nil || len(page) > 1 {
		if err == nil {
			err = fmt.Errorf("member pagination limit was ignored")
		}
		return []error{err}
	}
	return nil
}

func (h *Harness) protocolDiscoveryScenario(ctx context.Context) []error {
	a, err := h.discovery("agent-a")
	if err != nil {
		return []error{err}
	}
	doc, err := v2Call(a, "DiscoverProtocol", func() (contract.DiscoveryDocument, error) { return a.discovery.DiscoverProtocol(ctx) })
	if err != nil || doc.Protocol == "" || doc.Version == "" || len(doc.Capabilities) == 0 || len(doc.Transports) == 0 || len(doc.AuthMethods) == 0 || len(doc.SchemaURLs) == 0 || len(doc.UsageHints) == 0 {
		if err == nil {
			err = fmt.Errorf("protocol discovery document is incomplete")
		}
		return []error{err}
	}
	for _, operation := range requiredV2Operations {
		if !containsOperation(doc.Operations, operation) {
			return []error{fmt.Errorf("discovery omitted required V2 operation %q", operation)}
		}
	}
	schema, err := v2Call(a, "GetSchema", func() (contract.SchemaDocument, error) { return a.discovery.GetSchema(ctx, "harnesstalkie/v2") })
	if err != nil {
		return []error{err}
	}
	if err = contract.ValidateSchemaDocument(schema, doc.Operations); err != nil {
		return []error{fmt.Errorf("machine-readable schema is invalid: %w", err)}
	}
	help, err := v2Call(a, "GetHelp", func() (contract.HelpDocument, error) { return a.discovery.GetHelp(ctx) })
	if err != nil || len(help.Commands) == 0 {
		if err == nil {
			err = fmt.Errorf("help document is empty")
		}
		return []error{err}
	}
	helpText := strings.ToLower(strings.Join(help.Commands, " ") + " " + strings.Join(help.Examples, " "))
	if !strings.Contains(helpText, "join") || !strings.Contains(helpText, "dm") {
		return []error{fmt.Errorf("help did not describe basic collaboration")}
	}
	presets, err := v2Call(a, "ListPresets", func() ([]contract.Preset, error) { return a.discovery.ListPresets(ctx) })
	if err != nil {
		return []error{err}
	}
	if len(presets) > 0 {
		overrides := contract.Manifest{Server: h.serverID, Identity: contract.ManifestIdentity{Name: "preset-agent"}, Response: contract.ResponseOptions{Mode: "compact"}}
		presetResult, presetErr := v2Call(a, "ApplyPreset", func() (contract.PresetResult, error) {
			return a.discovery.ApplyPreset(ctx, presets[0].Name, overrides)
		})
		if presetErr != nil {
			return []error{presetErr}
		}
		if presetResult.Preset != presets[0].Name || presetResult.Effective.Kind != "Session" || presetResult.Effective.Server != h.serverID || presetResult.Effective.Identity.Name != "preset-agent" || presetResult.Effective.Response.Mode != "compact" {
			return []error{fmt.Errorf("preset application did not return its effective configuration")}
		}
	}
	transports, err := v2Call(a, "ListTransports", func() ([]contract.TransportCapability, error) { return a.discovery.ListTransports(ctx) })
	if err != nil || len(transports) == 0 {
		if err == nil {
			err = fmt.Errorf("transport capability list is empty")
		}
		return []error{err}
	}
	for _, transport := range transports {
		if transport.Name == "" || transport.Address == "" || !transport.Authenticated || !transport.SharedState {
			return []error{fmt.Errorf("transport %q omitted address, authentication, or shared-state guarantees", transport.Name)}
		}
	}
	h.t.Metric("advertised_transport_count", float64(len(transports)))
	return nil
}

func (h *Harness) transportScenario(ctx context.Context) []error {
	factory, ok := h.factory.(contract.TransportFactory)
	if !ok {
		return []error{UnsupportedError{Capability: "TransportFactory"}}
	}
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	if a.discovery == nil {
		return []error{UnsupportedError{Capability: "DiscoveryClient"}}
	}
	transports, err := v2Call(a, "ListTransportsForInterop", func() ([]contract.TransportCapability, error) { return a.discovery.ListTransports(ctx) })
	if err != nil {
		return []error{err}
	}
	var selected []contract.TransportCapability
	for _, x := range transports {
		if x.Read && x.Write && x.Authenticated && x.SharedState {
			selected = append(selected, x)
		}
	}
	if len(selected) < 2 {
		h.t.Metric("cross_transport_pairs_tested", 0)
		return nil
	}
	bIdentity := h.identities["agent-b"]
	bRaw, err := factory.NewTransport(ctx, selected[1].Name, &bIdentity)
	if err != nil {
		return []error{err}
	}
	b := newSession(bRaw, h.t)
	b.identity = h.identities["agent-b"]
	first, err := v2Call(a, "InteropReadA", func() (contract.Server, error) {
		return a.v2.GetServer(ctx, h.serverID, contract.ResponseOptions{Select: []string{"id", "name", "member_count"}})
	})
	if err != nil {
		return []error{err}
	}
	bv, ok := b.client.(contract.V2Client)
	if !ok {
		return []error{UnsupportedError{Capability: "V2Client on second transport"}}
	}
	second, err := v2Call(b, "InteropReadB", func() (contract.Server, error) {
		return bv.GetServer(ctx, h.serverID, contract.ResponseOptions{Select: []string{"id", "name", "member_count"}})
	})
	if err != nil || first.ID != second.ID || first.Name != second.Name {
		if err == nil {
			err = fmt.Errorf("transports observed different Server state")
		}
		return []error{err}
	}
	h.t.Metric("cross_transport_pairs_tested", 1)
	return nil
}

func (h *Harness) agentEfficiencyScenario(ctx context.Context) []error {
	start := time.Now()
	h.t.BeginAgentJob("address-to-first-collaboration")
	passed := false
	defer func() { h.t.EndAgentJob(passed) }()
	raw, err := h.factory.New(ctx, nil)
	if err != nil {
		return []error{err}
	}
	a := newSession(raw, h.t)
	if a.discovery == nil || a.v2 == nil {
		return []error{UnsupportedError{Capability: "self-discovering V2 client"}}
	}
	doc, err := v2Call(a, "JobDiscoverProtocol", func() (contract.DiscoveryDocument, error) {
		return a.discovery.DiscoverProtocol(ctx)
	})
	if err != nil || doc.Protocol == "" || len(doc.AuthMethods) == 0 {
		if err == nil {
			err = fmt.Errorf("address-only client could not discover protocol and authentication")
		}
		return []error{err}
	}
	name := h.label("new-agent")
	if _, err = a.CreateOrLoadIdentity(ctx, name); err != nil {
		return []error{err}
	}
	if err = a.PublishProfile(ctx, contract.Profile{DisplayName: name, Kind: "agent", Harness: "walkiebench", Capabilities: []string{"collaboration"}, CurrentWork: "first collaboration"}); err != nil {
		return []error{err}
	}
	servers, err := v2Call(a, "JobFindServers", func() ([]contract.Server, error) {
		return a.v2.DiscoverServers(ctx, contract.ServerQuery{Tag: "walkiebench", JoinPolicy: contract.JoinPublic, Limit: 20, Response: contract.ResponseOptions{Mode: "compact", Select: []string{"id", "name", "join_policy"}}})
	})
	if err != nil || len(servers) == 0 {
		if err == nil {
			err = fmt.Errorf("agent efficiency job could not discover a public Server")
		}
		return []error{err}
	}
	server := servers[0]
	for _, candidate := range servers {
		if candidate.ID == h.serverID {
			server = candidate
			break
		}
	}
	if err = v2Err(a, "JobJoinServer", func() error { return a.v2.JoinServer(ctx, server.ID) }); err != nil {
		return []error{err}
	}
	peers, err := v2Call(a, "JobFindPeer", func() ([]contract.ServerMember, error) {
		return a.v2.FindServerMembers(ctx, contract.ServerMemberQuery{ServerID: server.ID, Capability: "collaboration", Online: boolPtr(true), Limit: 10, Response: contract.ResponseOptions{Mode: "compact", Select: []string{"id", "display_name", "capabilities"}}})
	})
	if err != nil {
		return []error{err}
	}
	var peer string
	for _, p := range peers {
		if p.IdentityID != a.identityID() && p.IdentityID == h.identities["agent-b"].ID {
			peer = p.IdentityID
			break
		}
	}
	if peer == "" {
		return []error{fmt.Errorf("agent efficiency job found no relevant peer")}
	}
	body := "first-collaboration-" + h.label("message")
	if _, err = a.SendDMWithOptions(ctx, contract.SendDMRequest{To: peer, Content: body, ClientMessageID: h.label("client-message")}); err != nil {
		return []error{err}
	}
	receiver := h.sessions["agent-b"]
	if receiver != nil {
		deadline := time.Now().Add(h.cfg.MaxDeliveryLatency)
		for time.Now().Before(deadline) {
			messages, x := receiver.ReceiveDMs(ctx)
			if x != nil {
				return []error{x}
			}
			if hasContent(messages, body) {
				reply := "reply-" + body
				if _, x = receiver.SendDM(ctx, a.identityID(), reply); x != nil {
					return []error{x}
				}
				replyDeadline := time.Now().Add(h.cfg.MaxDeliveryLatency)
				for time.Now().Before(replyDeadline) {
					inbox, y := a.ReceiveDMs(ctx)
					if y != nil {
						return []error{y}
					}
					if hasContent(inbox, reply) {
						h.t.SetTimeToFirstCollaboration(float64(time.Since(start).Microseconds()) / 1000)
						h.t.Metric("time_to_first_collaboration_ms", float64(time.Since(start).Microseconds())/1000)
						h.t.Metric("job_a_find_and_message", 1)
						passed = true
						return nil
					}
					time.Sleep(10 * time.Millisecond)
				}
				return []error{fmt.Errorf("agent efficiency job did not receive a reply")}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	return []error{fmt.Errorf("agent efficiency job did not receive a reply path")}
}

func (h *Harness) v2ScaleScenario(ctx context.Context) []error {
	a, err := h.v2("agent-a")
	if err != nil {
		return []error{err}
	}
	count := h.cfg.Workers * 3
	if count < 10 {
		count = 10
	}
	start := time.Now()
	for i := 0; i < count; i++ {
		if _, err = v2Call(a, "ScaleCreateServer", func() (contract.Server, error) {
			return a.v2.CreateServer(ctx, h.serverSpec(h.label(fmt.Sprintf("scale-%03d", i)), contract.JoinPublic))
		}); err != nil {
			return []error{err}
		}
	}
	h.t.Metric("v2_server_count", float64(count))
	h.t.Metric("v2_server_create_rate", float64(count)/time.Since(start).Seconds())
	listStart := time.Now()
	servers, err := v2Call(a, "ScaleListServers", func() ([]contract.Server, error) {
		return a.v2.ListServers(ctx, contract.ServerQuery{Tag: "walkiebench", Limit: count + 20, Response: contract.ResponseOptions{Mode: "compact", Select: []string{"id", "name", "join_policy"}}})
	})
	if err != nil {
		return []error{err}
	}
	h.t.Metric("large_server_lookup_latency_ms", float64(time.Since(listStart).Microseconds())/1000)
	if len(servers) < count {
		return []error{fmt.Errorf("large Server lookup returned %d of at least %d Servers", len(servers), count)}
	}
	return nil
}

func boolPtr(v bool) *bool { return &v }

func stringPtr(v string) *string { return &v }
