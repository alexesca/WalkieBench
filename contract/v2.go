package contract

import (
	"context"
	"fmt"
	"strings"
)

// JoinPolicy is observable enrollment policy. Implementations may realize it
// however they choose; these values are the stable behavior vocabulary.
type JoinPolicy string

const (
	JoinPublic   JoinPolicy = "public"
	JoinApproval JoinPolicy = "approval-required"
	JoinInvite   JoinPolicy = "invite-only"
	JoinClosed   JoinPolicy = "closed"
)

type Visibility string

const (
	VisibilityServer  Visibility = "server-wide"
	VisibilityGroup   Visibility = "group-only"
	VisibilityShared  Visibility = "directly-shared"
	VisibilityPrivate Visibility = "private"
)

type MemberRole string

const (
	RoleOwner     MemberRole = "owner"
	RoleAdmin     MemberRole = "administrator"
	RoleModerator MemberRole = "moderator"
	RoleMember    MemberRole = "member"
	RoleGuest     MemberRole = "guest"
	RoleAgent     MemberRole = "agent"
)

type GroupRole string

const (
	GroupRoleOwner  GroupRole = "owner"
	GroupRoleAdmin  GroupRole = "administrator"
	GroupRoleMember GroupRole = "member"
)

const (
	GroupPermissionView       = "view_group"
	GroupPermissionHistory    = "view_history"
	GroupPermissionSend       = "send_messages"
	GroupPermissionInvite     = "invite_members"
	GroupPermissionApprove    = "approve_members"
	GroupPermissionRemove     = "remove_members"
	GroupPermissionManage     = "manage_settings"
	GroupPermissionManageRole = "manage_roles"
)

const (
	PermissionViewServer         = "view_server"
	PermissionViewMembers        = "view_members"
	PermissionSendMessages       = "send_messages"
	PermissionInviteMembers      = "invite_members"
	PermissionApproveMembers     = "approve_members"
	PermissionRemoveMembers      = "remove_members"
	PermissionCreateGroups       = "create_groups"
	PermissionManageGroups       = "manage_groups"
	PermissionCreatePosts        = "create_posts"
	PermissionModeratePosts      = "moderate_posts"
	PermissionManageRoles        = "manage_roles"
	PermissionManagePermissions  = "manage_permissions"
	PermissionManageSettings     = "manage_settings"
	PermissionViewAdministration = "view_administration"
	PermissionViewAudit          = "view_audit"
)

var RequiredServerPermissions = []string{
	PermissionViewServer, PermissionViewMembers, PermissionSendMessages,
	PermissionInviteMembers, PermissionApproveMembers, PermissionRemoveMembers,
	PermissionCreateGroups, PermissionManageGroups, PermissionCreatePosts,
	PermissionModeratePosts, PermissionManageRoles, PermissionManagePermissions,
	PermissionManageSettings, PermissionViewAdministration, PermissionViewAudit,
}

type ResponseOptions struct {
	Select      []string `json:"select,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	Cursor      uint64   `json:"cursor,omitempty"`
	UnreadOnly  bool     `json:"unread_only,omitempty"`
	SinceCursor uint64   `json:"since_cursor,omitempty"`
}

type Server struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Description       string     `json:"description,omitempty"`
	Purpose           string     `json:"purpose,omitempty"`
	Topics            []string   `json:"topics,omitempty"`
	Tags              []string   `json:"tags,omitempty"`
	OwnerID           string     `json:"owner_id"`
	JoinPolicy        JoinPolicy `json:"join_policy"`
	MemberCount       int        `json:"member_count"`
	Capabilities      []string   `json:"capabilities,omitempty"`
	Rules             []string   `json:"rules,omitempty"`
	Discoverable      bool       `json:"discoverable"`
	ConnectionMethods []string   `json:"connection_methods,omitempty"`
	Version           string     `json:"version,omitempty"`
}

type ServerSpec struct {
	Name              string     `json:"name"`
	Description       string     `json:"description,omitempty"`
	Purpose           string     `json:"purpose,omitempty"`
	Topics            []string   `json:"topics,omitempty"`
	Tags              []string   `json:"tags,omitempty"`
	JoinPolicy        JoinPolicy `json:"join_policy"`
	Capabilities      []string   `json:"capabilities,omitempty"`
	Rules             []string   `json:"rules,omitempty"`
	Discoverable      bool       `json:"discoverable"`
	ConnectionMethods []string   `json:"connection_methods,omitempty"`
}

type ServerPatch struct {
	Name         *string     `json:"name,omitempty"`
	Description  *string     `json:"description,omitempty"`
	Purpose      *string     `json:"purpose,omitempty"`
	Topics       []string    `json:"topics,omitempty"`
	Tags         []string    `json:"tags,omitempty"`
	JoinPolicy   *JoinPolicy `json:"join_policy,omitempty"`
	Rules        []string    `json:"rules,omitempty"`
	Discoverable *bool       `json:"discoverable,omitempty"`
}

type ServerQuery struct {
	Query      string          `json:"query,omitempty"`
	Topic      string          `json:"topic,omitempty"`
	Tag        string          `json:"tag,omitempty"`
	JoinPolicy JoinPolicy      `json:"join_policy,omitempty"`
	OwnerID    string          `json:"owner_id,omitempty"`
	Limit      int             `json:"limit,omitempty"`
	Cursor     uint64          `json:"cursor,omitempty"`
	Response   ResponseOptions `json:"response,omitempty"`
}

type ServerAccessRequest struct {
	ID        string `json:"id"`
	ServerID  string `json:"server_id"`
	Requester string `json:"requester"`
	Reason    string `json:"reason,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
}

type ServerInvite struct {
	ID        string `json:"id"`
	ServerID  string `json:"server_id"`
	InviteeID string `json:"invitee_id"`
	InvitedBy string `json:"invited_by"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type ServerMember struct {
	Participant
	ServerID    string     `json:"server_id"`
	Role        MemberRole `json:"role"`
	Permissions []string   `json:"permissions,omitempty"`
	JoinedAt    string     `json:"joined_at,omitempty"`
}

type ServerMemberQuery struct {
	ServerID    string          `json:"server_id"`
	Name        string          `json:"name,omitempty"`
	Capability  string          `json:"capability,omitempty"`
	Harness     string          `json:"harness,omitempty"`
	Topic       string          `json:"topic,omitempty"`
	CurrentWork string          `json:"current_work,omitempty"`
	Online      *bool           `json:"online,omitempty"`
	Limit       int             `json:"limit,omitempty"`
	Cursor      uint64          `json:"cursor,omitempty"`
	Response    ResponseOptions `json:"response,omitempty"`
}

type ServerRole struct {
	ServerID    string     `json:"server_id"`
	Participant string     `json:"participant"`
	Role        MemberRole `json:"role"`
	Permissions []string   `json:"permissions,omitempty"`
}

type PermissionChange struct {
	Role       MemberRole `json:"role"`
	Permission string     `json:"permission"`
	Allowed    bool       `json:"allowed"`
}

type GroupSpec struct {
	ServerID    string     `json:"server_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	JoinPolicy  JoinPolicy `json:"join_policy"`
	Private     bool       `json:"private,omitempty"`
}

type GroupPatch struct {
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	JoinPolicy  *JoinPolicy `json:"join_policy,omitempty"`
}

type GroupQuery struct {
	ServerID string          `json:"server_id"`
	Query    string          `json:"query,omitempty"`
	Limit    int             `json:"limit,omitempty"`
	Cursor   uint64          `json:"cursor,omitempty"`
	Response ResponseOptions `json:"response,omitempty"`
}

type GroupAccessRequest struct {
	ID        string `json:"id"`
	GroupID   string `json:"group_id"`
	Requester string `json:"requester"`
	Reason    string `json:"reason,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
}

type GroupInvite struct {
	ID        string `json:"id"`
	GroupID   string `json:"group_id"`
	InviteeID string `json:"invitee_id"`
	InvitedBy string `json:"invited_by"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type GroupMember struct {
	ServerMember
	GroupID          string    `json:"group_id"`
	GroupRole        GroupRole `json:"group_role"`
	GroupPermissions []string  `json:"group_permissions,omitempty"`
}

type GroupPermissionChange struct {
	Role       GroupRole `json:"role"`
	Permission string    `json:"permission"`
	Allowed    bool      `json:"allowed"`
}

type PostSpec struct {
	ServerID   string     `json:"server_id"`
	GroupID    string     `json:"group_id,omitempty"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Visibility Visibility `json:"visibility"`
	Mentions   []string   `json:"mentions,omitempty"`
	SharedWith []string   `json:"shared_with,omitempty"`
}

type PostPatch struct {
	Title      *string     `json:"title,omitempty"`
	Content    *string     `json:"content,omitempty"`
	Visibility *Visibility `json:"visibility,omitempty"`
}

type PostQuery struct {
	ServerID   string          `json:"server_id"`
	GroupID    string          `json:"group_id,omitempty"`
	Query      string          `json:"query,omitempty"`
	Visibility Visibility      `json:"visibility,omitempty"`
	Limit      int             `json:"limit,omitempty"`
	Cursor     uint64          `json:"cursor,omitempty"`
	Response   ResponseOptions `json:"response,omitempty"`
}

type PostView struct {
	Post
	ServerID   string     `json:"server_id"`
	GroupID    string     `json:"group_id,omitempty"`
	Visibility Visibility `json:"visibility"`
	Mentions   []string   `json:"mentions,omitempty"`
	SharedWith []string   `json:"shared_with,omitempty"`
}

type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	ActorID   string `json:"actor_id"`
	TargetID  string `json:"target_id"`
	ServerID  string `json:"server_id,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Sequence  uint64 `json:"sequence"`
	CreatedAt string `json:"created_at,omitempty"`
	Read      bool   `json:"read"`
}

const (
	NotificationDM                = "dm"
	NotificationMention           = "mention"
	NotificationReply             = "reply"
	NotificationReaction          = "reaction"
	NotificationInvitation        = "invitation"
	NotificationMembershipRequest = "membership_request"
	NotificationRequestApproved   = "request_approved"
	NotificationRequestRejected   = "request_rejected"
	NotificationAdminActivity     = "admin_activity"
)

type NotificationQuery struct {
	AfterSequence uint64          `json:"after_sequence,omitempty"`
	Limit         int             `json:"limit,omitempty"`
	UnreadOnly    bool            `json:"unread_only,omitempty"`
	Response      ResponseOptions `json:"response,omitempty"`
}

type Manifest struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Server     string             `json:"server"`
	Identity   ManifestIdentity   `json:"identity"`
	Membership ManifestMembership `json:"membership,omitempty"`
	Discover   ManifestDiscovery  `json:"discover,omitempty"`
	Groups     ManifestGroups     `json:"groups,omitempty"`
	Contacts   []string           `json:"contacts,omitempty"`
	Follows    []string           `json:"follows,omitempty"`
	Sync       ManifestSync       `json:"sync,omitempty"`
	Presence   ManifestPresence   `json:"presence,omitempty"`
	Response   ResponseOptions    `json:"response,omitempty"`
}

type ManifestIdentity struct {
	Name    string   `json:"name"`
	Profile *Profile `json:"profile,omitempty"`
}
type ManifestMembership struct {
	Join              string `json:"join,omitempty"`
	RequestIfRequired bool   `json:"requestIfRequired,omitempty"`
	AcceptInvitation  bool   `json:"acceptInvitation,omitempty"`
}
type ManifestDiscovery struct {
	Capabilities []string `json:"capabilities,omitempty"`
	Harness      string   `json:"harness,omitempty"`
	Limit        int      `json:"limit,omitempty"`
}
type ManifestGroups struct {
	Discover   bool `json:"discover,omitempty"`
	JoinPublic bool `json:"joinPublic,omitempty"`
	Limit      int  `json:"limit,omitempty"`
}
type ManifestSync struct {
	Inbox    bool   `json:"inbox,omitempty"`
	Mentions bool   `json:"mentions,omitempty"`
	Since    string `json:"since,omitempty"`
}
type ManifestPresence struct {
	Online bool `json:"online"`
}

type ManifestResult struct {
	Identity       Identity             `json:"identity"`
	Server         Server               `json:"server"`
	Membership     string               `json:"membership"`
	AccessRequest  *ServerAccessRequest `json:"access_request,omitempty"`
	Participants   []Participant        `json:"participants,omitempty"`
	Groups         []Group              `json:"groups,omitempty"`
	Contacts       []Contact            `json:"contacts,omitempty"`
	Follows        []string             `json:"follows,omitempty"`
	UnreadActivity int                  `json:"unread_activity"`
	Cursor         uint64               `json:"cursor"`
}

type BatchOperation struct {
	ID        string         `json:"id"`
	Operation string         `json:"operation"`
	Params    map[string]any `json:"params,omitempty"`
	DependsOn []string       `json:"depends_on,omitempty"`
}
type BatchRequest struct {
	Operations []BatchOperation `json:"operations"`
	Mode       string           `json:"mode,omitempty"`
}
type BatchResult struct {
	ID      string         `json:"id"`
	Success bool           `json:"success"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *ContractError `json:"error,omitempty"`
}
type BatchResponse struct {
	Results    []BatchResult `json:"results"`
	RoundTrips int           `json:"round_trips"`
}
type ContractError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable"`
	Action      string `json:"action,omitempty"`
}

type SyncRequest struct {
	ServerID    string          `json:"server_id"`
	AfterCursor uint64          `json:"after_cursor,omitempty"`
	Limit       int             `json:"limit,omitempty"`
	Include     []string        `json:"include,omitempty"`
	Response    ResponseOptions `json:"response,omitempty"`
}
type Delta struct {
	Cursor        uint64                `json:"cursor"`
	Events        []ActivityEvent       `json:"events,omitempty"`
	Notifications []Notification        `json:"notifications,omitempty"`
	Members       []ServerMember        `json:"members,omitempty"`
	Groups        []Group               `json:"groups,omitempty"`
	Posts         []PostView            `json:"posts,omitempty"`
	Comments      []CommentNode         `json:"comments,omitempty"`
	Messages      []Message             `json:"messages,omitempty"`
	Invites       []ServerInvite        `json:"invites,omitempty"`
	Requests      []ServerAccessRequest `json:"requests,omitempty"`
	More          bool                  `json:"more"`
}

type DiscoveryDocument struct {
	Protocol      string   `json:"protocol"`
	Version       string   `json:"version"`
	Capabilities  []string `json:"capabilities"`
	Operations    []string `json:"operations"`
	Transports    []string `json:"transports"`
	SchemaURLs    []string `json:"schema_urls,omitempty"`
	Documentation []string `json:"documentation,omitempty"`
	AuthMethods   []string `json:"auth_methods,omitempty"`
	UsageHints    []string `json:"usage_hints,omitempty"`
}
type SchemaDocument struct {
	Name       string                     `json:"name"`
	Version    string                     `json:"version"`
	Schema     any                        `json:"schema"`
	Operations map[string]OperationSchema `json:"operations"`
}
type OperationSchema struct {
	Description  string `json:"description,omitempty"`
	AuthRequired bool   `json:"auth_required"`
	Params       any    `json:"params"`
	Result       any    `json:"result"`
}
type HelpDocument struct {
	Commands []string `json:"commands"`
	Examples []string `json:"examples,omitempty"`
	Manifest string   `json:"manifest,omitempty"`
	Batch    string   `json:"batch,omitempty"`
}
type Preset struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Manifest    Manifest `json:"manifest"`
}
type PresetResult struct {
	Preset    string   `json:"preset"`
	Effective Manifest `json:"effective"`
}
type TransportCapability struct {
	Name          string `json:"name"`
	Read          bool   `json:"read"`
	Write         bool   `json:"write"`
	Streaming     bool   `json:"streaming"`
	Authenticated bool   `json:"authenticated"`
	SharedState   bool   `json:"shared_state"`
	Address       string `json:"address,omitempty"`
}

// V2Client is optional so a V1 implementation can return explicit unsupported
// capability results without forcing the benchmark to depend on its internals.
type V2Client interface {
	CreateServer(context.Context, ServerSpec) (Server, error)
	GetServer(context.Context, string, ResponseOptions) (Server, error)
	UpdateServer(context.Context, string, ServerPatch) (Server, error)
	ListServers(context.Context, ServerQuery) ([]Server, error)
	DiscoverServers(context.Context, ServerQuery) ([]Server, error)
	JoinServer(context.Context, string) error
	RequestServerAccess(context.Context, string, string) (ServerAccessRequest, error)
	ApproveServerRequest(context.Context, string) error
	RejectServerRequest(context.Context, string) error
	InviteToServer(context.Context, string, string) (ServerInvite, error)
	AcceptServerInvite(context.Context, string) error
	LeaveServer(context.Context, string) error
	RemoveServerMember(context.Context, string, string) error
	ListServerMembers(context.Context, ServerMemberQuery) ([]ServerMember, error)
	GetServerMember(context.Context, string, string, ResponseOptions) (ServerMember, error)
	ListServerRequests(context.Context, string, ResponseOptions) ([]ServerAccessRequest, error)
	ListServerInvites(context.Context, string, ResponseOptions) ([]ServerInvite, error)
	FindServerMembers(context.Context, ServerMemberQuery) ([]ServerMember, error)
	ListServerRoles(context.Context, string) ([]ServerRole, error)
	SetServerRole(context.Context, string, string, MemberRole) error
	UpdateServerPermissions(context.Context, string, PermissionChange) error
	GetServerAudit(context.Context, string, ResponseOptions) ([]ActivityEvent, error)

	CreateServerGroup(context.Context, GroupSpec) (Group, error)
	UpdateGroupV2(context.Context, string, GroupPatch) (Group, error)
	DeleteGroupV2(context.Context, string) error
	DiscoverGroups(context.Context, GroupQuery) ([]Group, error)
	ListServerGroups(context.Context, GroupQuery) ([]Group, error)
	JoinGroupV2(context.Context, string) error
	RequestGroupAccess(context.Context, string, string) (GroupAccessRequest, error)
	ApproveGroupRequest(context.Context, string) error
	RejectGroupRequest(context.Context, string) error
	InviteToGroup(context.Context, string, string) (GroupInvite, error)
	AcceptGroupInvite(context.Context, string) error
	LeaveGroupV2(context.Context, string) error
	RemoveGroupMember(context.Context, string, string) error
	ListGroupMembers(context.Context, string, ResponseOptions) ([]GroupMember, error)
	ListGroupRequests(context.Context, string, ResponseOptions) ([]GroupAccessRequest, error)
	ListGroupInvites(context.Context, string, ResponseOptions) ([]GroupInvite, error)
	SetGroupRole(context.Context, string, string, GroupRole) error
	UpdateGroupPermissions(context.Context, string, GroupPermissionChange) error

	CreateServerPost(context.Context, PostSpec) (PostView, error)
	EditServerPost(context.Context, string, PostPatch) (PostView, error)
	CommentServer(context.Context, string, string, []string) (CommentNode, error)
	GetServerThread(context.Context, string, ResponseOptions) (Thread, error)
	DiscoverPosts(context.Context, PostQuery) ([]PostView, error)
	SearchPosts(context.Context, PostQuery) ([]PostView, error)
	SharePost(context.Context, string, string, string) error
	ListNotifications(context.Context, NotificationQuery) ([]Notification, error)
	MarkNotificationsRead(context.Context, []string) error

	ApplyManifest(context.Context, Manifest) (ManifestResult, error)
	Batch(context.Context, BatchRequest) (BatchResponse, error)
	Sync(context.Context, SyncRequest) (Delta, error)
}

type DiscoveryClient interface {
	DiscoverProtocol(context.Context) (DiscoveryDocument, error)
	GetSchema(context.Context, string) (SchemaDocument, error)
	GetHelp(context.Context) (HelpDocument, error)
	ListPresets(context.Context) ([]Preset, error)
	ApplyPreset(context.Context, string, Manifest) (PresetResult, error)
	ListTransports(context.Context) ([]TransportCapability, error)
}

type TransportFactory interface {
	NewTransport(context.Context, string, *Identity) (Client, error)
}

func ValidateManifest(m Manifest) error {
	if m.APIVersion != "harnesstalkie/v2" {
		return fmt.Errorf("apiVersion must be harnesstalkie/v2")
	}
	if m.Kind != "Session" {
		return fmt.Errorf("kind must be Session")
	}
	if m.Server == "" {
		return fmt.Errorf("server is required")
	}
	if m.Identity.Name == "" {
		return fmt.Errorf("identity.name is required")
	}
	if m.Membership.Join != "" && m.Membership.Join != "if-allowed" && m.Membership.Join != "always" {
		return fmt.Errorf("membership.join must be if-allowed or always")
	}
	if m.Sync.Since != "" && m.Sync.Since != "last" && m.Sync.Since != "beginning" {
		return fmt.Errorf("sync.since must be last or beginning")
	}
	if m.Discover.Limit < 0 || m.Groups.Limit < 0 || m.Response.Limit < 0 {
		return fmt.Errorf("limits cannot be negative")
	}
	if m.Response.Mode != "" && m.Response.Mode != "compact" && m.Response.Mode != "full" {
		return fmt.Errorf("response.mode must be compact or full")
	}
	return nil
}

func ValidateBatch(b BatchRequest) error { _, err := BatchOrder(b.Operations); return err }

func BatchOrder(ops []BatchOperation) ([]BatchOperation, error) {
	byID := map[string]BatchOperation{}
	state := map[string]int{}
	for _, op := range ops {
		if op.ID == "" || op.Operation == "" {
			return nil, fmt.Errorf("batch operations require id and operation")
		}
		if _, ok := byID[op.ID]; ok {
			return nil, fmt.Errorf("duplicate batch operation %q", op.ID)
		}
		byID[op.ID] = op
		if err := validateResultReferences(op.Params); err != nil {
			return nil, fmt.Errorf("operation %q: %w", op.ID, err)
		}
	}
	for _, op := range ops {
		deps := map[string]bool{}
		for _, dep := range op.DependsOn {
			deps[dep] = true
		}
		for _, ref := range resultReferences(op.Params) {
			if _, ok := byID[ref]; !ok {
				return nil, fmt.Errorf("operation %q references unknown result %q", op.ID, ref)
			}
			if !deps[ref] {
				return nil, fmt.Errorf("operation %q must depend on referenced result %q", op.ID, ref)
			}
		}
	}
	var out []BatchOperation
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("batch dependency cycle at %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		op, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown dependency %q", id)
		}
		state[id] = 1
		for _, dep := range op.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		out = append(out, op)
		return nil
	}
	for _, op := range ops {
		if err := visit(op.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func validateResultReferences(v any) error {
	switch x := v.(type) {
	case map[string]any:
		for _, child := range x {
			if err := validateResultReferences(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range x {
			if err := validateResultReferences(child); err != nil {
				return err
			}
		}
	case string:
		if strings.HasPrefix(x, "$ref:") {
			parts := strings.Split(strings.TrimPrefix(x, "$ref:"), ".")
			if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("invalid result reference %q", x)
			}
		}
	}
	return nil
}

func resultReferences(v any) []string {
	var out []string
	switch x := v.(type) {
	case map[string]any:
		for _, child := range x {
			out = append(out, resultReferences(child)...)
		}
	case []any:
		for _, child := range x {
			out = append(out, resultReferences(child)...)
		}
	case string:
		if strings.HasPrefix(x, "$ref:") {
			out = append(out, strings.Split(strings.TrimPrefix(x, "$ref:"), ".")[0])
		}
	}
	return out
}

func ValidateSchemaDocument(doc SchemaDocument, advertised []string) error {
	if doc.Name == "" || doc.Version == "" {
		return fmt.Errorf("schema name and version are required")
	}
	root, ok := doc.Schema.(map[string]any)
	if !ok || root["type"] != "object" {
		return fmt.Errorf("schema root must be an object schema")
	}
	if len(doc.Operations) == 0 {
		return fmt.Errorf("operation schemas are required")
	}
	for _, name := range advertised {
		op, ok := doc.Operations[name]
		if !ok {
			return fmt.Errorf("advertised operation %q has no schema", name)
		}
		if op.Params == nil || op.Result == nil {
			return fmt.Errorf("operation %q must describe params and result", name)
		}
	}
	return nil
}
