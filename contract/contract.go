// Package contract contains the stable, implementation-independent WalkieBench surface.
package contract

import (
	"context"
	"time"
)

type Identity struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	SessionToken string `json:"session_token,omitempty"`
}

type Profile struct {
	DisplayName         string   `json:"display_name"`
	Bio                 string   `json:"bio,omitempty"`
	Repository          string   `json:"repository,omitempty"`
	Harness             string   `json:"harness,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
	CurrentWork         string   `json:"current_work,omitempty"`
	Limitations         string   `json:"limitations,omitempty"`
	CollaborationTopics []string `json:"collaboration_topics,omitempty"`
}

type Presence struct {
	IdentityID string    `json:"identity_id"`
	Online     bool      `json:"online"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Contact struct {
	IdentityID  string `json:"identity_id"`
	DisplayName string `json:"display_name,omitempty"`
}

type Participant struct {
	IdentityID          string    `json:"identity_id"`
	Handle              string    `json:"handle,omitempty"`
	DisplayName         string    `json:"display_name"`
	Bio                 string    `json:"bio,omitempty"`
	Repository          string    `json:"repository,omitempty"`
	Harness             string    `json:"harness,omitempty"`
	Capabilities        []string  `json:"capabilities,omitempty"`
	CurrentWork         string    `json:"current_work,omitempty"`
	Limitations         string    `json:"limitations,omitempty"`
	CollaborationTopics []string  `json:"collaboration_topics,omitempty"`
	Online              bool      `json:"online"`
	LastSeen            time.Time `json:"last_seen"`
}

type ParticipantQuery struct {
	Query       string `json:"query,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Harness     string `json:"harness,omitempty"`
	Capability  string `json:"capability,omitempty"`
	CurrentWork string `json:"current_work,omitempty"`
}

type Invitation struct {
	Group     Group     `json:"group"`
	InvitedBy string    `json:"invited_by"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID              string    `json:"id"`
	SenderID        string    `json:"sender_id"`
	RecipientID     string    `json:"recipient_id,omitempty"`
	GroupID         string    `json:"group_id,omitempty"`
	Content         string    `json:"content"`
	Sequence        uint64    `json:"sequence"`
	CreatedAt       time.Time `json:"created_at"`
	Read            bool      `json:"read,omitempty"`
	ConversationID  string    `json:"conversation_id,omitempty"`
	ReplyTo         string    `json:"reply_to,omitempty"`
	ClientMessageID string    `json:"client_message_id,omitempty"`
}

type SendDMRequest struct {
	To              string `json:"to"`
	Content         string `json:"content"`
	ReplyTo         string `json:"reply_to,omitempty"`
	ClientMessageID string `json:"client_message_id,omitempty"`
}

type MessageQuery struct {
	With          string `json:"with,omitempty"`
	AfterSequence uint64 `json:"after_sequence,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	UnreadOnly    bool   `json:"unread_only,omitempty"`
}

type MessagePage struct {
	Messages   []Message `json:"messages"`
	NextCursor uint64    `json:"next_cursor"`
	More       bool      `json:"more"`
}

type EventQuery struct {
	AfterSequence uint64 `json:"after_sequence,omitempty"`
	WaitMS        int    `json:"wait_ms,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Ack           bool   `json:"ack,omitempty"`
}

type ActivityEvent struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	ActorID   string    `json:"actor_id"`
	TargetID  string    `json:"target_id,omitempty"`
	Sequence  uint64    `json:"sequence"`
	CreatedAt time.Time `json:"created_at"`
	Summary   string    `json:"summary,omitempty"`
	Message   *Message  `json:"message,omitempty"`
}

type EventBatch struct {
	Events     []ActivityEvent `json:"events"`
	NextCursor uint64          `json:"next_cursor"`
	More       bool            `json:"more"`
}

type Bootstrap struct {
	Identity     Identity      `json:"identity"`
	Participants []Participant `json:"participants,omitempty"`
	Invites      []Invitation  `json:"invites,omitempty"`
	Groups       []Group       `json:"groups,omitempty"`
	RecentPosts  []Post        `json:"recent_posts,omitempty"`
	UnreadDMs    int           `json:"unread_dms"`
	Cursor       uint64        `json:"cursor"`
}

type Group struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

type Post struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CommentNode struct {
	ID        string         `json:"id"`
	AuthorID  string         `json:"author_id"`
	ParentID  string         `json:"parent_id,omitempty"`
	Content   string         `json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	Children  []*CommentNode `json:"children,omitempty"`
}

type Thread struct {
	Post      Post           `json:"post"`
	Comments  []*CommentNode `json:"comments"`
	Followers []string       `json:"followers"`
	Reactions map[string]int `json:"reactions,omitempty"`
}

type ResumeResult struct {
	RestoredMessages int `json:"restored_messages"`
	RestoredThreads  int `json:"restored_threads"`
}

// Client is the only application surface used by benchmark scenarios.
// A new Client represents a new harness/session. Implementations persist identity
// material outside the connection and must accept the identity returned by the
// previous session in CreateOrLoadIdentity.
type Client interface {
	CreateOrLoadIdentity(context.Context, string) (Identity, error)
	PublishProfile(context.Context, Profile) error
	GetPresence(context.Context, string) (Presence, error)
	ListOnline(context.Context) ([]Presence, error)
	ConnectTo(context.Context, string) error
	ListContacts(context.Context) ([]Contact, error)
	ListParticipants(context.Context, ParticipantQuery) ([]Participant, error)
	FindPeers(context.Context, ParticipantQuery) ([]Participant, error)
	ListInvites(context.Context) ([]Invitation, error)
	ListGroups(context.Context) ([]Group, error)
	ListPublicPosts(context.Context) ([]Post, error)
	GetCapabilities(context.Context) ([]string, error)
	Heartbeat(context.Context) error
	Bootstrap(context.Context, bool, bool, bool) (Bootstrap, error)
	ConnectAndBootstrap(context.Context, string) (Bootstrap, error)
	WaitForEvents(context.Context, EventQuery) (EventBatch, error)

	SendDM(context.Context, string, string) (Message, error)
	SendDMWithOptions(context.Context, SendDMRequest) (Message, error)
	GetDMHistory(context.Context, string) ([]Message, error)
	GetDMHistoryPage(context.Context, MessageQuery) (MessagePage, error)
	ReceiveDMs(context.Context) ([]Message, error)
	ReceiveDMsPage(context.Context, MessageQuery) (MessagePage, error)
	MarkRead(context.Context, []string) error

	CreateGroup(context.Context, string) (Group, error)
	Invite(context.Context, string, string) error
	Join(context.Context, string) error
	Leave(context.Context, string) error
	SendGroupMessage(context.Context, string, string) (Message, error)
	GetGroupHistory(context.Context, string) ([]Message, error)

	CreatePost(context.Context, string, string) (Post, error)
	Comment(context.Context, string, string) (CommentNode, error)
	GetThread(context.Context, string) (Thread, error)
	FollowThread(context.Context, string) error
	UnfollowThread(context.Context, string) error
	React(context.Context, string, string) error

	Resume(context.Context) (ResumeResult, error)
	Close() error
}
