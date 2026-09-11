package benchmark

import (
	"context"
	"fmt"

	"walkiebench/contract"
	"walkiebench/telemetry"
)

var requiredV2Operations = []string{
	"CreateServer", "GetServer", "UpdateServer", "ListServers", "DiscoverServers",
	"JoinServer", "RequestServerAccess", "ApproveServerRequest", "RejectServerRequest",
	"InviteToServer", "AcceptServerInvite", "LeaveServer", "RemoveServerMember",
	"ListServerMembers", "GetServerMember", "ListServerRequests", "ListServerInvites",
	"FindServerMembers", "ListServerRoles", "SetServerRole", "UpdateServerPermissions", "GetServerAudit",
	"CreateGroup", "UpdateGroup", "DeleteGroup", "DiscoverGroups", "ListGroups",
	"JoinGroup", "RequestGroupAccess", "ApproveGroupRequest", "RejectGroupRequest",
	"InviteToGroup", "AcceptGroupInvite", "LeaveGroup", "RemoveGroupMember",
	"ListGroupMembers", "ListGroupRequests", "ListGroupInvites", "SetGroupRole", "UpdateGroupPermissions",
	"CreatePost", "EditPost", "Comment", "GetThread", "DiscoverPosts", "SearchPosts", "SharePost",
	"ListNotifications", "MarkNotificationsRead", "ApplyManifest", "Batch", "Sync",
}

// granularPermissionScenario proves that permissions change behavior. Merely
// returning role names or accepting a permission-update call is not enough.
func (h *Harness) granularPermissionScenario(ctx context.Context) []error {
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

	server, err := v2Call(a, "PermissionFixtureCreateServer", func() (contract.Server, error) {
		return a.v2.CreateServer(ctx, h.serverSpec(h.label("permission-matrix"), contract.JoinPublic))
	})
	if err != nil {
		return []error{err}
	}
	for _, member := range []*Session{b, c} {
		if err = v2Err(member, "PermissionFixtureJoin", func() error { return member.v2.JoinServer(ctx, server.ID) }); err != nil {
			return []error{err}
		}
	}
	if err = v2Err(a, "PermissionFixtureMemberRole", func() error {
		return a.v2.SetServerRole(ctx, server.ID, b.identityID(), contract.RoleMember)
	}); err != nil {
		return []error{err}
	}
	if err = v2Err(a, "PermissionFixtureModeratorRole", func() error {
		return a.v2.SetServerRole(ctx, server.ID, c.identityID(), contract.RoleModerator)
	}); err != nil {
		return []error{err}
	}

	set := func(role contract.MemberRole, permission string, allowed bool) error {
		return a.v2.UpdateServerPermissions(ctx, server.ID, contract.PermissionChange{Role: role, Permission: permission, Allowed: allowed})
	}
	expectDenied := func(label string, action func() error) error {
		if action() == nil {
			h.t.AddReliability(func(r *telemetry.Reliability) { r.AccessControlViolations++ })
			return fmt.Errorf("%s succeeded without its granular permission", label)
		}
		return nil
	}

	if err = expectDenied("member Server update", func() error {
		name := h.label("unauthorized-update")
		_, e := b.v2.UpdateServer(ctx, server.ID, contract.ServerPatch{Name: &name})
		return e
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionManageSettings, true); err != nil {
		return []error{err}
	}
	updatedName := h.label("authorized-update")
	if _, err = v2Call(b, "PermissionManageSettingsAllowed", func() (contract.Server, error) {
		return b.v2.UpdateServer(ctx, server.ID, contract.ServerPatch{Name: &updatedName})
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionManageSettings, false); err != nil {
		return []error{err}
	}

	if err = expectDenied("member group creation", func() error {
		_, e := b.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: server.ID, Name: h.label("forbidden-group"), JoinPolicy: contract.JoinPublic})
		return e
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionCreateGroups, true); err != nil {
		return []error{err}
	}
	group, err := v2Call(b, "PermissionCreateGroupsAllowed", func() (contract.Group, error) {
		return b.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: server.ID, Name: h.label("allowed-group"), JoinPolicy: contract.JoinPublic})
	})
	if err != nil || group.ServerID != server.ID || group.OwnerID != b.identityID() {
		if err == nil {
			err = fmt.Errorf("group authorization returned incomplete scoped state")
		}
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionCreateGroups, false); err != nil {
		return []error{err}
	}

	if err = expectDenied("member post creation", func() error {
		_, e := b.v2.CreateServerPost(ctx, contract.PostSpec{ServerID: server.ID, Title: "forbidden", Content: "forbidden", Visibility: contract.VisibilityServer})
		return e
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionCreatePosts, true); err != nil {
		return []error{err}
	}
	post, err := v2Call(b, "PermissionCreatePostsAllowed", func() (contract.PostView, error) {
		return b.v2.CreateServerPost(ctx, contract.PostSpec{ServerID: server.ID, Title: h.label("member-post"), Content: "member content", Visibility: contract.VisibilityServer})
	})
	if err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionCreatePosts, false); err != nil {
		return []error{err}
	}

	if err = expectDenied("member audit view", func() error {
		_, e := b.v2.GetServerAudit(ctx, server.ID, contract.ResponseOptions{Limit: 5})
		return e
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionViewAudit, true); err != nil {
		return []error{err}
	}
	if audit, e := b.v2.GetServerAudit(ctx, server.ID, contract.ResponseOptions{Limit: 5}); e != nil || len(audit) == 0 {
		if e == nil {
			e = fmt.Errorf("view_audit permission returned no audit state")
		}
		return []error{e}
	}

	if err = expectDenied("member role management", func() error {
		return b.v2.SetServerRole(ctx, server.ID, c.identityID(), contract.RoleMember)
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionManageRoles, true); err != nil {
		return []error{err}
	}
	if err = b.v2.SetServerRole(ctx, server.ID, c.identityID(), contract.RoleModerator); err != nil {
		return []error{err}
	}
	if err = expectDenied("role manager permission management", func() error {
		return b.v2.UpdateServerPermissions(ctx, server.ID, contract.PermissionChange{Role: contract.RoleModerator, Permission: contract.PermissionModeratePosts, Allowed: true})
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionManagePermissions, true); err != nil {
		return []error{err}
	}
	if err = b.v2.UpdateServerPermissions(ctx, server.ID, contract.PermissionChange{Role: contract.RoleModerator, Permission: contract.PermissionModeratePosts, Allowed: true}); err != nil {
		return []error{err}
	}
	if _, err = c.v2.EditServerPost(ctx, post.ID, contract.PostPatch{Title: stringPtr("moderated title")}); err != nil {
		return []error{fmt.Errorf("granted moderation permission had no effect: %w", err)}
	}

	if err = expectDenied("member invitation", func() error {
		_, e := b.v2.InviteToServer(ctx, server.ID, human.identityID())
		return e
	}); err != nil {
		return []error{err}
	}
	if err = set(contract.RoleMember, contract.PermissionInviteMembers, true); err != nil {
		return []error{err}
	}
	invite, err := b.v2.InviteToServer(ctx, server.ID, human.identityID())
	if err != nil || invite.InvitedBy != b.identityID() {
		if err == nil {
			err = fmt.Errorf("invite_members did not retain the authorized inviter")
		}
		return []error{err}
	}

	if err = set(contract.RoleOwner, contract.PermissionManageSettings, false); err != nil {
		return []error{err}
	}
	ownerName := h.label("owner-supreme")
	if _, err = a.v2.UpdateServer(ctx, server.ID, contract.ServerPatch{Name: &ownerName}); err != nil {
		return []error{fmt.Errorf("owner lost supreme Server authority: %w", err)}
	}
	return nil
}

func (h *Harness) notificationDurabilityScenario(ctx context.Context) []error {
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

	server, err := a.v2.CreateServer(ctx, h.serverSpec(h.label("notification-matrix"), contract.JoinPublic))
	if err != nil {
		return []error{err}
	}
	for _, member := range []*Session{b, c} {
		if err = member.v2.JoinServer(ctx, server.ID); err != nil {
			return []error{err}
		}
	}
	clear := func(s *Session) error {
		ns, e := s.v2.ListNotifications(ctx, contract.NotificationQuery{Limit: 500})
		if e != nil {
			return e
		}
		ids := make([]string, 0, len(ns))
		for _, n := range ns {
			ids = append(ids, n.ID)
		}
		if len(ids) > 0 {
			return s.v2.MarkNotificationsRead(ctx, ids)
		}
		return nil
	}
	for _, s := range []*Session{a, b, c, human} {
		if err = clear(s); err != nil {
			return []error{err}
		}
	}

	dmBody := "notification-dm-" + h.label("body")
	if _, err = a.SendDMWithOptions(ctx, contract.SendDMRequest{To: b.identityID(), Content: dmBody, ClientMessageID: h.label("notification-dm")}); err != nil {
		return []error{err}
	}
	post, err := b.v2.CreateServerPost(ctx, contract.PostSpec{ServerID: server.ID, Title: h.label("notification-post"), Content: "notification matrix", Visibility: contract.VisibilityServer, Mentions: []string{c.identityID()}})
	if err != nil {
		return []error{err}
	}
	if err = b.FollowThread(ctx, post.ID); err != nil {
		return []error{err}
	}
	if _, err = c.v2.CommentServer(ctx, post.ID, "followed reply", nil); err != nil {
		return []error{err}
	}
	if err = a.React(ctx, post.ID, "like"); err != nil {
		return []error{err}
	}
	group, err := a.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: server.ID, Name: h.label("notification-group"), JoinPolicy: contract.JoinInvite})
	if err != nil {
		return []error{err}
	}
	if _, err = a.v2.InviteToGroup(ctx, group.ID, b.identityID()); err != nil {
		return []error{err}
	}
	if _, err = a.v2.InviteToServer(ctx, server.ID, human.identityID()); err != nil {
		return []error{err}
	}
	approval, err := a.v2.CreateServer(ctx, h.serverSpec(h.label("notification-approval"), contract.JoinApproval))
	if err != nil {
		return []error{err}
	}
	request, err := c.v2.RequestServerAccess(ctx, approval.ID, "notification request")
	if err != nil {
		return []error{err}
	}
	if err = a.v2.ApproveServerRequest(ctx, request.ID); err != nil {
		return []error{err}
	}

	expect := func(s *Session, typ, target string) ([]contract.Notification, error) {
		ns, e := s.v2.ListNotifications(ctx, contract.NotificationQuery{UnreadOnly: true, Limit: 500, Response: contract.ResponseOptions{Mode: "compact"}})
		if e != nil {
			return nil, e
		}
		if !containsNotification(ns, typ, target) {
			return ns, fmt.Errorf("durable notification %q for %q was not delivered", typ, target)
		}
		return ns, nil
	}
	bNotifications, err := expect(b, contract.NotificationDM, "")
	if err != nil {
		return []error{err}
	}
	for _, check := range []struct {
		typ    string
		target string
	}{{contract.NotificationReply, post.ID}, {contract.NotificationReaction, post.ID}, {contract.NotificationInvitation, group.ID}} {
		if !containsNotification(bNotifications, check.typ, check.target) {
			return []error{fmt.Errorf("notification matrix omitted %s for %s", check.typ, check.target)}
		}
	}
	if _, err = expect(c, contract.NotificationMention, post.ID); err != nil {
		return []error{err}
	}
	if _, err = expect(human, contract.NotificationInvitation, server.ID); err != nil {
		return []error{err}
	}
	if _, err = expect(a, contract.NotificationMembershipRequest, request.ID); err != nil {
		return []error{err}
	}
	if _, err = expect(c, contract.NotificationRequestApproved, request.ID); err != nil {
		return []error{err}
	}

	beforeIDs := map[string]bool{}
	for _, n := range bNotifications {
		beforeIDs[n.ID] = true
	}
	reconnected, err := h.reconnect(ctx, "agent-b")
	if err != nil {
		return []error{err}
	}
	afterReconnect, err := reconnected.v2.ListNotifications(ctx, contract.NotificationQuery{UnreadOnly: true, Limit: 500})
	if err != nil {
		return []error{err}
	}
	for id := range beforeIDs {
		found := false
		for _, n := range afterReconnect {
			found = found || n.ID == id
		}
		if !found {
			return []error{fmt.Errorf("notification %s was lost across reconnect", id)}
		}
	}
	ids := make([]string, 0, len(afterReconnect))
	for _, n := range afterReconnect {
		ids = append(ids, n.ID)
	}
	if err = reconnected.v2.MarkNotificationsRead(ctx, ids); err != nil {
		return []error{err}
	}
	unread, err := reconnected.v2.ListNotifications(ctx, contract.NotificationQuery{UnreadOnly: true, Limit: 500})
	if err != nil {
		return []error{err}
	}
	for _, n := range unread {
		if beforeIDs[n.ID] {
			return []error{fmt.Errorf("read notification %s remained in unread-only results", n.ID)}
		}
	}
	return nil
}

func (h *Harness) securityIsolationScenario(ctx context.Context) []error {
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

	isolated, err := c.v2.CreateServer(ctx, h.serverSpec(h.label("isolated-server"), contract.JoinPublic))
	if err != nil {
		return []error{err}
	}
	privateGroup, err := c.v2.CreateServerGroup(ctx, contract.GroupSpec{ServerID: isolated.ID, Name: h.label("isolated-group"), JoinPolicy: contract.JoinClosed, Private: true})
	if err != nil {
		return []error{err}
	}
	marker := "cross-server-secret-" + h.label("marker")
	privatePost, err := c.v2.CreateServerPost(ctx, contract.PostSpec{ServerID: isolated.ID, GroupID: privateGroup.ID, Title: h.label("isolated-post"), Content: marker, Visibility: contract.VisibilityGroup})
	if err != nil {
		return []error{err}
	}

	deny := func(label string, action func() error) error {
		if action() == nil {
			h.t.AddReliability(func(r *telemetry.Reliability) { r.AccessControlViolations++ })
			return fmt.Errorf("%s bypassed authorization", label)
		}
		return nil
	}
	checks := []struct {
		label  string
		action func() error
	}{
		{"cross-Server member enumeration", func() error {
			_, e := b.v2.ListServerMembers(ctx, contract.ServerMemberQuery{ServerID: isolated.ID})
			return e
		}},
		{"cross-Server group discovery", func() error { _, e := b.v2.DiscoverGroups(ctx, contract.GroupQuery{ServerID: isolated.ID}); return e }},
		{"cross-group thread access", func() error { _, e := b.v2.GetServerThread(ctx, privatePost.ID, contract.ResponseOptions{}); return e }},
		{"cross-Server search", func() error {
			_, e := b.v2.SearchPosts(ctx, contract.PostQuery{ServerID: isolated.ID, Query: marker})
			return e
		}},
		{"cross-Server scoped DM", func() error {
			_, e := b.SendDMWithOptions(ctx, contract.SendDMRequest{ServerID: isolated.ID, To: c.identityID(), Content: marker})
			return e
		}},
		{"role manipulation", func() error { return b.v2.SetServerRole(ctx, isolated.ID, b.identityID(), contract.RoleAdmin) }},
		{"cursor access", func() error { _, e := b.v2.Sync(ctx, contract.SyncRequest{ServerID: isolated.ID}); return e }},
	}
	for _, check := range checks {
		if err = deny(check.label, check.action); err != nil {
			return []error{err}
		}
	}

	invite, err := c.v2.InviteToServer(ctx, isolated.ID, human.identityID())
	if err != nil {
		return []error{err}
	}
	if err = deny("forged invite acceptance", func() error { return b.v2.AcceptServerInvite(ctx, invite.ID) }); err != nil {
		return []error{err}
	}
	approval, err := c.v2.CreateServer(ctx, h.serverSpec(h.label("forged-approval"), contract.JoinApproval))
	if err != nil {
		return []error{err}
	}
	request, err := human.v2.RequestServerAccess(ctx, approval.ID, "security-negative")
	if err != nil {
		return []error{err}
	}
	if err = deny("forged approval", func() error { return b.v2.ApproveServerRequest(ctx, request.ID) }); err != nil {
		return []error{err}
	}

	batch, err := b.v2.Batch(ctx, contract.BatchRequest{Mode: "continue", Operations: []contract.BatchOperation{
		{ID: "public", Operation: "DiscoverServers", Params: map[string]any{"limit": 1}},
		{ID: "forbidden", Operation: "ListServerMembers", Params: map[string]any{"server_id": isolated.ID}},
	}})
	if err != nil {
		return []error{err}
	}
	forbidden := false
	for _, result := range batch.Results {
		if result.ID == "forbidden" && !result.Success && result.Error != nil {
			forbidden = true
		}
	}
	if !forbidden {
		return []error{fmt.Errorf("batch authorization bypass did not return a scoped operation error")}
	}

	compact, err := b.v2.GetServer(ctx, isolated.ID, contract.ResponseOptions{Mode: "compact", Select: []string{"id", "name", "join_policy"}})
	if err != nil {
		return []error{err}
	}
	if compact.ID != isolated.ID || compact.Name != isolated.Name || compact.JoinPolicy != isolated.JoinPolicy {
		return []error{fmt.Errorf("compact public Server metadata lost selected semantics")}
	}
	if compact.Description != "" || compact.Purpose != "" || compact.OwnerID != "" || len(compact.Rules) != 0 {
		return []error{fmt.Errorf("compact Server response leaked unselected metadata")}
	}

	if _, err = c.v2.Sync(ctx, contract.SyncRequest{ServerID: isolated.ID, AfterCursor: ^uint64(0), Limit: 10}); err == nil {
		// A cursor beyond the current head is malformed/stale, not a valid way to
		// move the synchronization cursor backwards.
		return []error{fmt.Errorf("future cursor was accepted without a recovery error")}
	}
	badManifest := contract.Manifest{APIVersion: "harnesstalkie/v2", Kind: "Session", Server: isolated.ID, Identity: contract.ManifestIdentity{Name: c.identity.DisplayName}, Membership: contract.ManifestMembership{Join: "bypass"}}
	if err = deny("malformed manifest", func() error { _, e := c.v2.ApplyManifest(ctx, badManifest); return e }); err != nil {
		return []error{err}
	}
	return nil
}
