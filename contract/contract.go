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
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio,omitempty"`
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

type Message struct {
	ID          string    `json:"id"`
	SenderID    string    `json:"sender_id"`
	RecipientID string    `json:"recipient_id,omitempty"`
	GroupID     string    `json:"group_id,omitempty"`
	Content     string    `json:"content"`
	Sequence    uint64    `json:"sequence"`
	CreatedAt   time.Time `json:"created_at"`
	Read        bool      `json:"read,omitempty"`
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

	SendDM(context.Context, string, string) (Message, error)
	GetDMHistory(context.Context, string) ([]Message, error)
	ReceiveDMs(context.Context) ([]Message, error)
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
