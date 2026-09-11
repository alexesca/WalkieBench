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
		h.t.RegisterPlaintext(value)
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
	if public.ID == "" || public.OwnerID != a.identityID() || public.JoinPolicy != contract.JoinPublic {
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
	if err != nil || req.ID == "" {
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
	if err != nil || len(requests) == 0 {
		if err == nil {
			err = fmt.Errorf("owner could not see pending access request")
		}
		return []error{err}
	}
	if err = v2Err(a, "ApproveServerRequest", func() error { return a.v2.ApproveServerRequest(ctx, req.ID) }); err != nil {
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
	if err != nil || inv.ID == "" {
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
	if err = v2Err(c, "LeaveServer", func() error { return c.v2.LeaveServer(ctx, inviteOnly.ID) }); err != nil {
		return []error{err}
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
	if err = v2Err(a, "RemoveServerMember", func() error { return a.v2.RemoveServerMember(ctx, approval.ID, c.identityID()) }); err != nil {
		return []error{err}
	}
	publicPolicy := contract.JoinPublic
	updated, err := v2Call(a, "UpdateServer", func() (contract.Server, error) {
		return a.v2.UpdateServer(ctx, approval.ID, contract.ServerPatch{JoinPolicy: &publicPolicy})
	})
	if err != nil || updated.JoinPolicy != contract.JoinPublic {
		if err == nil {
			err = fmt.Errorf("Server join-policy transition was not visible")
		}
		return []error{err}
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
	members, err := v2Call(a, "ListServerMembers", func() ([]contract.ServerMember, error) {
		return a.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID, Capability: "collaboration", Limit: 20, Response: contract.ResponseOptions{Mode: "compact"}})
	})
	if err != nil || !containsMember(members, b.identityID()) {
		if err == nil {
			err = fmt.Errorf("authorized member discovery omitted agent-b")
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
	if err = v2Err(a, "UpdateGroup", func() error {
		name := h.label("renamed-group")
		_, e := a.v2.UpdateGroupV2(ctx, public.ID, contract.GroupPatch{Name: &name})
		return e
	}); err != nil {
		return []error{err}
	}
	members, err := v2Call(a, "ListGroupMembers", func() ([]contract.ServerMember, error) {
		return a.v2.ListGroupMembers(ctx, public.ID, contract.ResponseOptions{Limit: 10})
	})
	if err != nil || !containsMember(members, b.identityID()) {
		if err == nil {
			err = fmt.Errorf("group member listing omitted joined member")
		}
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
	manifest := contract.Manifest{APIVersion: "harnesstalkie/v2", Kind: "Session", Server: h.serverID, Identity: contract.ManifestIdentity{Name: "agent-a", Profile: &contract.Profile{DisplayName: "agent-a", Harness: "benchmark", Capabilities: []string{"collaboration"}}}, Membership: contract.ManifestMembership{Join: "if-allowed", RequestIfRequired: true, AcceptInvitation: true}, Discover: contract.ManifestDiscovery{Capabilities: []string{"collaboration"}, Limit: 5}, Sync: contract.ManifestSync{Inbox: true, Mentions: true, Since: "last"}, Presence: contract.ManifestPresence{Online: true}, Response: contract.ResponseOptions{Mode: "compact", Select: []string{"id", "name", "status", "capabilities"}}}
	h.probe("collaboration")
	if err = contract.ValidateManifest(manifest); err != nil {
		return []error{err}
	}
	first, err := v2Call(a, "ApplyManifest", func() (contract.ManifestResult, error) { return a.v2.ApplyManifest(ctx, manifest) })
	if err != nil {
		return []error{err}
	}
	second, err := v2Call(a, "ApplyManifestIdempotent", func() (contract.ManifestResult, error) { return a.v2.ApplyManifest(ctx, manifest) })
	if err != nil || first.Server.ID != second.Server.ID || first.Identity.ID != second.Identity.ID {
		if err == nil {
			err = fmt.Errorf("repeated manifest changed desired identity or Server")
		}
		return []error{err}
	}
	before, err := v2Call(a, "ListMembersBeforeManifest", func() ([]contract.ServerMember, error) {
		return a.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID})
	})
	if err != nil {
		return []error{err}
	}
	after, err := v2Call(a, "ListMembersAfterManifest", func() ([]contract.ServerMember, error) {
		return a.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID})
	})
	if err != nil || len(after) != len(before) {
		if err == nil {
			err = fmt.Errorf("manifest application duplicated membership")
		}
		return []error{err}
	}
	h.t.Metric("job_f_declarative_bootstrap", 1)
	batch := contract.BatchRequest{Mode: "ordered", Operations: []contract.BatchOperation{
		{ID: "servers", Operation: "ListServers", Params: map[string]any{"limit": 5}},
		{ID: "members", Operation: "ListServerMembers", DependsOn: []string{"servers"}, Params: map[string]any{"server_id": h.serverID, "limit": 5}},
	}}
	if err = contract.ValidateBatch(batch); err != nil {
		return []error{err}
	}
	primitiveStart := time.Now()
	if _, err = v2Call(a, "PrimitiveWorkflowListServers", func() ([]contract.Server, error) {
		return a.v2.ListServers(ctx, contract.ServerQuery{Limit: 5})
	}); err != nil {
		return []error{err}
	}
	if _, err = v2Call(a, "PrimitiveWorkflowListMembers", func() ([]contract.ServerMember, error) {
		return a.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: h.serverID, Limit: 5})
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
	if err != nil || len(response.Results) != 2 || response.Results[0].ID != "servers" || response.Results[1].ID != "members" {
		if err == nil {
			err = fmt.Errorf("batch response did not preserve dependency order")
		}
		return []error{err}
	}
	failedBatch, batchErr := v2Call(a, "BatchPartialFailure", func() (contract.BatchResponse, error) {
		return a.v2.Batch(ctx, contract.BatchRequest{Mode: "ordered", Operations: []contract.BatchOperation{{ID: "allowed", Operation: "ListServers", Params: map[string]any{"limit": 1}}, {ID: "forbidden", Operation: "ListServerMembers", DependsOn: []string{"allowed"}, Params: map[string]any{"server_id": "server-does-not-exist"}}}})
	})
	if batchErr == nil {
		partialFailure := false
		for _, result := range failedBatch.Results {
			if result.ID == "forbidden" && !result.Success {
				partialFailure = true
			}
		}
		if !partialFailure {
			return []error{fmt.Errorf("batch accepted an unauthorized dependent operation")}
		}
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
	if err != nil || doc.Protocol == "" || doc.Version == "" || len(doc.Capabilities) == 0 || len(doc.Transports) == 0 {
		if err == nil {
			err = fmt.Errorf("protocol discovery document is incomplete")
		}
		return []error{err}
	}
	if !containsOperation(doc.Operations, "CreateServer") || !containsOperation(doc.Operations, "ApplyManifest") {
		return []error{fmt.Errorf("discovery omitted required V2 operations")}
	}
	schema, err := v2Call(a, "GetSchema", func() (contract.SchemaDocument, error) { return a.discovery.GetSchema(ctx, "harnesstalkie/v2") })
	if err != nil {
		return []error{err}
	}
	if schema.Name == "" || schema.Schema == nil {
		return []error{fmt.Errorf("machine-readable schema is empty")}
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
		presetResult, presetErr := v2Call(a, "ApplyPreset", func() (contract.PresetResult, error) { return a.discovery.ApplyPreset(ctx, presets[0].Name) })
		if presetErr != nil {
			return []error{presetErr}
		}
		if presetResult.Preset != presets[0].Name || presetResult.Effective.Kind != "Session" {
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
		if x.Read && x.Write {
			selected = append(selected, x)
		}
	}
	if len(selected) < 2 {
		return []error{UnsupportedError{Capability: "two read/write transports"}}
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
	return nil
}

func (h *Harness) agentEfficiencyScenario(ctx context.Context) []error {
	start := time.Now()
	a, err := h.v2("agent-a")
	if err != nil {
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
		if p.IdentityID != a.identityID() {
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
	peerName := "agent-b"
	for name, id := range h.identities {
		if id.ID == peer {
			peerName = name
		}
	}
	receiver := h.sessions[peerName]
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
