package transport

import (
	"context"

	"walkiebench/contract"
)

func (c *JSONRPCClient) CreateServer(ctx context.Context, spec contract.ServerSpec) (contract.Server, error) {
	var v contract.Server
	e := c.call(ctx, "CreateServer", spec, &v)
	return v, e
}
func (c *JSONRPCClient) GetServer(ctx context.Context, id string, o contract.ResponseOptions) (contract.Server, error) {
	var v contract.Server
	e := c.call(ctx, "GetServer", map[string]any{"server_id": id, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) UpdateServer(ctx context.Context, id string, p contract.ServerPatch) (contract.Server, error) {
	var v contract.Server
	e := c.call(ctx, "UpdateServer", map[string]any{"server_id": id, "patch": p}, &v)
	return v, e
}
func (c *JSONRPCClient) ListServers(ctx context.Context, q contract.ServerQuery) ([]contract.Server, error) {
	var v []contract.Server
	e := c.call(ctx, "ListServers", q, &v)
	return v, e
}
func (c *JSONRPCClient) DiscoverServers(ctx context.Context, q contract.ServerQuery) ([]contract.Server, error) {
	var v []contract.Server
	e := c.call(ctx, "DiscoverServers", q, &v)
	return v, e
}
func (c *JSONRPCClient) JoinServer(ctx context.Context, id string) error {
	return c.call(ctx, "JoinServer", map[string]string{"server_id": id}, nil)
}
func (c *JSONRPCClient) RequestServerAccess(ctx context.Context, id, reason string) (contract.ServerAccessRequest, error) {
	var v contract.ServerAccessRequest
	e := c.call(ctx, "RequestServerAccess", map[string]string{"server_id": id, "reason": reason}, &v)
	return v, e
}
func (c *JSONRPCClient) ApproveServerRequest(ctx context.Context, id string) error {
	return c.call(ctx, "ApproveServerRequest", map[string]string{"request_id": id}, nil)
}
func (c *JSONRPCClient) RejectServerRequest(ctx context.Context, id string) error {
	return c.call(ctx, "RejectServerRequest", map[string]string{"request_id": id}, nil)
}
func (c *JSONRPCClient) InviteToServer(ctx context.Context, server, participant string) (contract.ServerInvite, error) {
	var v contract.ServerInvite
	e := c.call(ctx, "InviteToServer", map[string]string{"server_id": server, "participant_id": participant}, &v)
	return v, e
}
func (c *JSONRPCClient) AcceptServerInvite(ctx context.Context, id string) error {
	return c.call(ctx, "AcceptServerInvite", map[string]string{"invite_id": id}, nil)
}
func (c *JSONRPCClient) LeaveServer(ctx context.Context, id string) error {
	return c.call(ctx, "LeaveServer", map[string]string{"server_id": id}, nil)
}
func (c *JSONRPCClient) RemoveServerMember(ctx context.Context, server, participant string) error {
	return c.call(ctx, "RemoveServerMember", map[string]string{"server_id": server, "participant_id": participant}, nil)
}
func (c *JSONRPCClient) ListServerMembers(ctx context.Context, q contract.ServerMemberQuery) ([]contract.ServerMember, error) {
	var v []contract.ServerMember
	e := c.call(ctx, "ListServerMembers", q, &v)
	return v, e
}
func (c *JSONRPCClient) GetServerMember(ctx context.Context, server, participant string, o contract.ResponseOptions) (contract.ServerMember, error) {
	var v contract.ServerMember
	e := c.call(ctx, "GetServerMember", map[string]any{"server_id": server, "participant_id": participant, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) ListServerRequests(ctx context.Context, server string, o contract.ResponseOptions) ([]contract.ServerAccessRequest, error) {
	var v []contract.ServerAccessRequest
	e := c.call(ctx, "ListServerRequests", map[string]any{"server_id": server, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) ListServerInvites(ctx context.Context, server string, o contract.ResponseOptions) ([]contract.ServerInvite, error) {
	var v []contract.ServerInvite
	e := c.call(ctx, "ListServerInvites", map[string]any{"server_id": server, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) FindServerMembers(ctx context.Context, q contract.ServerMemberQuery) ([]contract.ServerMember, error) {
	var v []contract.ServerMember
	e := c.call(ctx, "FindServerMembers", q, &v)
	return v, e
}
func (c *JSONRPCClient) ListServerRoles(ctx context.Context, server string) ([]contract.ServerRole, error) {
	var v []contract.ServerRole
	e := c.call(ctx, "ListServerRoles", map[string]string{"server_id": server}, &v)
	return v, e
}
func (c *JSONRPCClient) SetServerRole(ctx context.Context, server, participant string, role contract.MemberRole) error {
	return c.call(ctx, "SetServerRole", map[string]any{"server_id": server, "participant_id": participant, "role": role}, nil)
}
func (c *JSONRPCClient) UpdateServerPermissions(ctx context.Context, server string, p contract.PermissionChange) error {
	return c.call(ctx, "UpdateServerPermissions", map[string]any{"server_id": server, "change": p}, nil)
}
func (c *JSONRPCClient) GetServerAudit(ctx context.Context, server string, o contract.ResponseOptions) ([]contract.ActivityEvent, error) {
	var v []contract.ActivityEvent
	e := c.call(ctx, "GetServerAudit", map[string]any{"server_id": server, "response": o}, &v)
	return v, e
}

func (c *JSONRPCClient) CreateServerGroup(ctx context.Context, spec contract.GroupSpec) (contract.Group, error) {
	var v contract.Group
	e := c.call(ctx, "CreateGroup", spec, &v)
	return v, e
}
func (c *JSONRPCClient) UpdateGroupV2(ctx context.Context, id string, p contract.GroupPatch) (contract.Group, error) {
	var v contract.Group
	e := c.call(ctx, "UpdateGroup", map[string]any{"group_id": id, "patch": p}, &v)
	return v, e
}
func (c *JSONRPCClient) DeleteGroupV2(ctx context.Context, id string) error {
	return c.call(ctx, "DeleteGroup", map[string]string{"group_id": id}, nil)
}
func (c *JSONRPCClient) DiscoverGroups(ctx context.Context, q contract.GroupQuery) ([]contract.Group, error) {
	var v []contract.Group
	e := c.call(ctx, "DiscoverGroups", q, &v)
	return v, e
}
func (c *JSONRPCClient) ListServerGroups(ctx context.Context, q contract.GroupQuery) ([]contract.Group, error) {
	var v []contract.Group
	e := c.call(ctx, "ListGroups", q, &v)
	return v, e
}
func (c *JSONRPCClient) JoinGroupV2(ctx context.Context, id string) error {
	return c.call(ctx, "JoinGroup", map[string]string{"group_id": id}, nil)
}
func (c *JSONRPCClient) RequestGroupAccess(ctx context.Context, id, reason string) (contract.GroupAccessRequest, error) {
	var v contract.GroupAccessRequest
	e := c.call(ctx, "RequestGroupAccess", map[string]string{"group_id": id, "reason": reason}, &v)
	return v, e
}
func (c *JSONRPCClient) ApproveGroupRequest(ctx context.Context, id string) error {
	return c.call(ctx, "ApproveGroupRequest", map[string]string{"request_id": id}, nil)
}
func (c *JSONRPCClient) RejectGroupRequest(ctx context.Context, id string) error {
	return c.call(ctx, "RejectGroupRequest", map[string]string{"request_id": id}, nil)
}
func (c *JSONRPCClient) InviteToGroup(ctx context.Context, group, participant string) (contract.GroupInvite, error) {
	var v contract.GroupInvite
	e := c.call(ctx, "InviteToGroup", map[string]string{"group_id": group, "participant_id": participant}, &v)
	return v, e
}
func (c *JSONRPCClient) AcceptGroupInvite(ctx context.Context, id string) error {
	return c.call(ctx, "AcceptGroupInvite", map[string]string{"invite_id": id}, nil)
}
func (c *JSONRPCClient) LeaveGroupV2(ctx context.Context, id string) error {
	return c.call(ctx, "LeaveGroup", map[string]string{"group_id": id}, nil)
}
func (c *JSONRPCClient) RemoveGroupMember(ctx context.Context, group, participant string) error {
	return c.call(ctx, "RemoveGroupMember", map[string]string{"group_id": group, "participant_id": participant}, nil)
}
func (c *JSONRPCClient) ListGroupMembers(ctx context.Context, group string, o contract.ResponseOptions) ([]contract.GroupMember, error) {
	var v []contract.GroupMember
	e := c.call(ctx, "ListGroupMembers", map[string]any{"group_id": group, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) ListGroupRequests(ctx context.Context, group string, o contract.ResponseOptions) ([]contract.GroupAccessRequest, error) {
	var v []contract.GroupAccessRequest
	e := c.call(ctx, "ListGroupRequests", map[string]any{"group_id": group, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) ListGroupInvites(ctx context.Context, group string, o contract.ResponseOptions) ([]contract.GroupInvite, error) {
	var v []contract.GroupInvite
	e := c.call(ctx, "ListGroupInvites", map[string]any{"group_id": group, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) SetGroupRole(ctx context.Context, group, participant string, role contract.GroupRole) error {
	return c.call(ctx, "SetGroupRole", map[string]any{"group_id": group, "participant_id": participant, "role": role}, nil)
}
func (c *JSONRPCClient) UpdateGroupPermissions(ctx context.Context, group string, p contract.GroupPermissionChange) error {
	return c.call(ctx, "UpdateGroupPermissions", map[string]any{"group_id": group, "change": p}, nil)
}

func (c *JSONRPCClient) CreateServerPost(ctx context.Context, spec contract.PostSpec) (contract.PostView, error) {
	var v contract.PostView
	e := c.call(ctx, "CreatePost", spec, &v)
	return v, e
}
func (c *JSONRPCClient) EditServerPost(ctx context.Context, id string, p contract.PostPatch) (contract.PostView, error) {
	var v contract.PostView
	e := c.call(ctx, "EditPost", map[string]any{"post_id": id, "patch": p}, &v)
	return v, e
}
func (c *JSONRPCClient) CommentServer(ctx context.Context, target, content string, mentions []string) (contract.CommentNode, error) {
	var v contract.CommentNode
	e := c.call(ctx, "Comment", map[string]any{"post_or_comment": target, "content": content, "mentions": mentions}, &v)
	return v, e
}
func (c *JSONRPCClient) GetServerThread(ctx context.Context, id string, o contract.ResponseOptions) (contract.Thread, error) {
	var v contract.Thread
	e := c.call(ctx, "GetThread", map[string]any{"post_id": id, "response": o}, &v)
	return v, e
}
func (c *JSONRPCClient) DiscoverPosts(ctx context.Context, q contract.PostQuery) ([]contract.PostView, error) {
	var v []contract.PostView
	e := c.call(ctx, "DiscoverPosts", q, &v)
	return v, e
}
func (c *JSONRPCClient) SearchPosts(ctx context.Context, q contract.PostQuery) ([]contract.PostView, error) {
	var v []contract.PostView
	e := c.call(ctx, "SearchPosts", q, &v)
	return v, e
}
func (c *JSONRPCClient) SharePost(ctx context.Context, post, server, group string) error {
	return c.call(ctx, "SharePost", map[string]string{"post_id": post, "server_id": server, "group_id": group}, nil)
}
func (c *JSONRPCClient) ListNotifications(ctx context.Context, q contract.NotificationQuery) ([]contract.Notification, error) {
	var v []contract.Notification
	e := c.call(ctx, "ListNotifications", q, &v)
	return v, e
}
func (c *JSONRPCClient) MarkNotificationsRead(ctx context.Context, ids []string) error {
	return c.call(ctx, "MarkNotificationsRead", map[string]any{"notification_ids": ids}, nil)
}

func (c *JSONRPCClient) ApplyManifest(ctx context.Context, m contract.Manifest) (contract.ManifestResult, error) {
	var v contract.ManifestResult
	e := c.call(ctx, "ApplyManifest", m, &v)
	return v, e
}
func (c *JSONRPCClient) Batch(ctx context.Context, b contract.BatchRequest) (contract.BatchResponse, error) {
	var v contract.BatchResponse
	e := c.call(ctx, "Batch", b, &v)
	return v, e
}
func (c *JSONRPCClient) Sync(ctx context.Context, q contract.SyncRequest) (contract.Delta, error) {
	var v contract.Delta
	e := c.call(ctx, "Sync", q, &v)
	return v, e
}

func (c *JSONRPCClient) DiscoverProtocol(ctx context.Context) (contract.DiscoveryDocument, error) {
	var v contract.DiscoveryDocument
	e := c.call(ctx, "DiscoverProtocol", nil, &v)
	return v, e
}
func (c *JSONRPCClient) GetSchema(ctx context.Context, name string) (contract.SchemaDocument, error) {
	var v contract.SchemaDocument
	e := c.call(ctx, "GetSchema", map[string]string{"name": name}, &v)
	return v, e
}
func (c *JSONRPCClient) GetHelp(ctx context.Context) (contract.HelpDocument, error) {
	var v contract.HelpDocument
	e := c.call(ctx, "GetHelp", nil, &v)
	return v, e
}
func (c *JSONRPCClient) ListPresets(ctx context.Context) ([]contract.Preset, error) {
	var v []contract.Preset
	e := c.call(ctx, "ListPresets", nil, &v)
	return v, e
}
func (c *JSONRPCClient) ApplyPreset(ctx context.Context, name string, overrides contract.Manifest) (contract.PresetResult, error) {
	var v contract.PresetResult
	e := c.call(ctx, "ApplyPreset", map[string]any{"preset": name, "overrides": overrides}, &v)
	return v, e
}
func (c *JSONRPCClient) ListTransports(ctx context.Context) ([]contract.TransportCapability, error) {
	var v []contract.TransportCapability
	e := c.call(ctx, "ListTransports", nil, &v)
	return v, e
}

var _ contract.V2Client = (*JSONRPCClient)(nil)
var _ contract.DiscoveryClient = (*JSONRPCClient)(nil)
