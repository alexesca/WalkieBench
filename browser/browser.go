// Package browser provides the optional Chromium path. It drives vercel
// agent-browser (or a compatible command) through its stable CLI.
package browser

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Driver interface {
	Open(context.Context, string) error
	Snapshot(context.Context) (string, error)
	URL(context.Context) (string, error)
	SetViewport(context.Context, int, int) error
	Click(context.Context, string) error
	Fill(context.Context, string, string) error
	Press(context.Context, string, string) error
	Text(context.Context, string) (string, error)
	AssertVisible(context.Context, string, string) error
	Close(context.Context) error
}
type AgentBrowser struct{ Command string }

func (a AgentBrowser) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, a.Command, args...)
	out, e := cmd.CombinedOutput()
	if e != nil {
		return string(out), fmt.Errorf("%s %v: %w", a.Command, strings.Join(args, " "), e)
	}
	return string(out), nil
}
func (a AgentBrowser) Open(ctx context.Context, u string) error {
	_, e := a.run(ctx, "open", u)
	return e
}
func (a AgentBrowser) Snapshot(ctx context.Context) (string, error) { return a.run(ctx, "snapshot") }
func (a AgentBrowser) URL(ctx context.Context) (string, error)      { return a.run(ctx, "get", "url") }
func (a AgentBrowser) SetViewport(ctx context.Context, width, height int) error {
	_, e := a.run(ctx, "set", "viewport", fmt.Sprint(width), fmt.Sprint(height))
	return e
}
func (a AgentBrowser) Click(ctx context.Context, s string) error {
	_, e := a.run(ctx, "click", s)
	return e
}
func (a AgentBrowser) Fill(ctx context.Context, s, v string) error {
	_, e := a.run(ctx, "fill", s, v)
	return e
}
func (a AgentBrowser) Press(ctx context.Context, s, k string) error {
	_, e := a.run(ctx, "press", s, k)
	return e
}
func (a AgentBrowser) Text(ctx context.Context, s string) (string, error) {
	return a.run(ctx, "get", "text", s)
}
func (a AgentBrowser) AssertVisible(ctx context.Context, s, needle string) error {
	v, e := a.Text(ctx, s)
	if e != nil {
		return e
	}
	if !strings.Contains(strings.ToLower(v), strings.ToLower(needle)) {
		return fmt.Errorf("selector %q does not contain %q; got %q", s, needle, v)
	}
	return nil
}
func (a AgentBrowser) Close(ctx context.Context) error { _, e := a.run(ctx, "close"); return e }

type Selectors struct {
	IdentityID      string `json:"identity_id"`
	IdentityLoad    string `json:"identity_load"`
	IdentityVisible string `json:"identity_visible"`
	DMRecipient     string `json:"dm_recipient"`
	DMContent       string `json:"dm_content"`
	DMSend          string `json:"dm_send"`
	DMVisible       string `json:"dm_visible"`
	GroupName       string `json:"group_name"`
	GroupCreate     string `json:"group_create"`
	GroupID         string `json:"group_id"`
	GroupMessage    string `json:"group_message"`
	GroupSend       string `json:"group_send"`
	GroupVisible    string `json:"group_visible"`
	PostTitle       string `json:"post_title"`
	PostContent     string `json:"post_content"`
	PostCreate      string `json:"post_create"`
	PostVisible     string `json:"post_visible"`
	ThreadID        string `json:"thread_id"`
	CommentContent  string `json:"comment_content"`
	CommentSend     string `json:"comment_send"`
	CommentVisible  string `json:"comment_visible"`
	ThreadVisible   string `json:"thread_visible"`
	ThreadStructure string `json:"thread_structure"`
	Notification    string `json:"notifications_visible"`
	OrderVisible    string `json:"ordered_activity_visible"`
	Follow          string `json:"follow"`
	React           string `json:"react"`
	Presence        string `json:"presence"`
	ServerID        string `json:"server_id"`
	ServerVisible   string `json:"server_visible"`
	MemberVisible   string `json:"server_members_visible"`
	RequestVisible  string `json:"server_requests_visible"`
	ApproveRequest  string `json:"server_approve_request"`
	RoleParticipant string `json:"server_role_participant"`
	RoleValue       string `json:"server_role_value"`
	RoleSave        string `json:"server_role_save"`
	GroupAdmin      string `json:"group_admin_visible"`
	Moderation      string `json:"moderation_visible"`
	AuditVisible    string `json:"audit_visible"`
	NavInbox        string `json:"nav_inbox"`
	NavServers      string `json:"nav_servers"`
	NavMembers      string `json:"nav_members"`
	NavGroups       string `json:"nav_groups"`
	NavForums       string `json:"nav_forums"`
	NavAdmin        string `json:"nav_admin"`
}
type Config struct {
	URL                  string
	Selectors            Selectors
	IdentityID           string
	ServerID             string
	RoleParticipant      string
	RequireApplicationIA bool
	Observer             func(string, time.Duration, error)
}

type observedDriver struct {
	Driver
	observer func(string, time.Duration, error)
}

func (d observedDriver) measure(name string, f func() error) error {
	start := time.Now()
	e := f()
	if d.observer != nil {
		d.observer(name, time.Since(start), e)
	}
	return e
}
func (d observedDriver) Open(c context.Context, u string) error {
	return d.measure("Open", func() error { return d.Driver.Open(c, u) })
}
func (d observedDriver) Click(c context.Context, s string) error {
	return d.measure("Click", func() error { return d.Driver.Click(c, s) })
}
func (d observedDriver) Fill(c context.Context, s, v string) error {
	return d.measure("Fill", func() error { return d.Driver.Fill(c, s, v) })
}
func (d observedDriver) Press(c context.Context, s, k string) error {
	return d.measure("Press", func() error { return d.Driver.Press(c, s, k) })
}
func (d observedDriver) Close(c context.Context) error {
	return d.measure("Close", func() error { return d.Driver.Close(c) })
}
func (d observedDriver) AssertVisible(c context.Context, s, n string) error {
	return d.measure("AssertVisible", func() error { return d.Driver.AssertVisible(c, s, n) })
}
func (d observedDriver) Snapshot(c context.Context) (string, error) {
	start := time.Now()
	v, e := d.Driver.Snapshot(c)
	if d.observer != nil {
		d.observer("Snapshot", time.Since(start), e)
	}
	return v, e
}
func (d observedDriver) URL(c context.Context) (string, error) {
	start := time.Now()
	v, e := d.Driver.URL(c)
	if d.observer != nil {
		d.observer("URL", time.Since(start), e)
	}
	return v, e
}
func (d observedDriver) SetViewport(c context.Context, width, height int) error {
	return d.measure("SetViewport", func() error { return d.Driver.SetViewport(c, width, height) })
}

func verifyApplicationIA(ctx context.Context, d Driver, baseURL string, selectors Selectors) error {
	snapshot, err := d.Snapshot(ctx)
	if err != nil {
		return err
	}
	lower := strings.ToLower(snapshot)
	for _, landmark := range []string{"navigation", "main", "heading"} {
		if !strings.Contains(lower, landmark) {
			return fmt.Errorf("accessibility snapshot omitted %s landmark or role", landmark)
		}
	}
	routes := []struct {
		selector string
		path     string
	}{{selectors.NavInbox, "/inbox"}, {selectors.NavServers, "/servers"}, {selectors.NavMembers, "/members"}, {selectors.NavGroups, "/groups"}, {selectors.NavForums, "/forums"}, {selectors.NavAdmin, "/settings"}}
	for _, route := range routes {
		if route.selector == "" {
			return fmt.Errorf("browser contract omitted required navigation selector for %s", route.path)
		}
		if err = d.Click(ctx, route.selector); err != nil {
			return err
		}
		current, urlErr := d.URL(ctx)
		if urlErr != nil {
			return urlErr
		}
		if !strings.Contains(current, route.path) {
			return fmt.Errorf("navigation to %s did not produce a bookmarkable route; got %s", route.path, strings.TrimSpace(current))
		}
	}
	if err = d.SetViewport(ctx, 390, 844); err != nil {
		return err
	}
	if _, err = d.Snapshot(ctx); err != nil {
		return fmt.Errorf("mobile accessibility snapshot: %w", err)
	}
	if err = d.SetViewport(ctx, 1440, 900); err != nil {
		return err
	}
	return d.Open(ctx, baseURL)
}

// RunHumanFlow performs all human-visible major actions. Selectors use
// data-testid values by default; a future UI may supply equivalent refs.
func RunHumanFlow(ctx context.Context, d Driver, c Config, groupID, postID, agentID string) ([]string, error) {
	d = observedDriver{Driver: d, observer: c.Observer}
	if err := d.Open(ctx, c.URL); err != nil {
		return nil, err
	}
	defer d.Close(context.Background())
	if c.RequireApplicationIA {
		if err := verifyApplicationIA(ctx, d, c.URL, c.Selectors); err != nil {
			return nil, err
		}
	}
	if c.Selectors.IdentityID != "" && c.Selectors.IdentityLoad != "" {
		if err := waitVisible(ctx, d, c.Selectors.IdentityID, ""); err != nil {
			return nil, err
		}
		if err := d.Fill(ctx, c.Selectors.IdentityID, c.IdentityID); err != nil {
			return nil, err
		}
		if err := d.Click(ctx, c.Selectors.IdentityLoad); err != nil {
			return nil, err
		}
		if c.Selectors.IdentityVisible != "" {
			if err := waitVisible(ctx, d, c.Selectors.IdentityVisible, "Connected"); err != nil {
				return nil, err
			}
		}
	}
	baseURL := strings.TrimRight(c.URL, "/")
	check := func(sel, needle string) error { return waitVisible(ctx, d, sel, needle) }
	if c.ServerID != "" {
		if err := d.Open(ctx, baseURL+"/servers/"+c.ServerID+"/overview"); err != nil {
			return nil, err
		}
		if c.Selectors.ServerVisible != "" {
			if err := check(c.Selectors.ServerVisible, c.ServerID); err != nil {
				return nil, err
			}
		}
	}
	if c.Selectors.Presence != "" {
		if err := check(c.Selectors.Presence, "online"); err != nil {
			return nil, err
		}
	}
	if c.Selectors.NavGroups != "" {
		if err := d.Click(ctx, c.Selectors.NavGroups); err != nil {
			return nil, err
		}
		if c.Selectors.GroupName != "" {
			if err := check(c.Selectors.GroupName, ""); err != nil {
				return nil, err
			}
		}
	}
	if c.Selectors.GroupName != "" && c.Selectors.GroupCreate != "" {
		if err := d.Fill(ctx, c.Selectors.GroupName, "human-ui-group"); err != nil {
			return nil, err
		}
		if err := d.Click(ctx, c.Selectors.GroupCreate); err != nil {
			return nil, err
		}
	}
	if err := d.Open(ctx, baseURL+"/dm/"+agentID); err != nil {
		return nil, err
	}
	if err := check(c.Selectors.DMRecipient, ""); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.DMRecipient, agentID); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.DMContent, "human-ui-dm"); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.DMSend); err != nil {
		return nil, err
	}
	if err := check(c.Selectors.DMVisible, "human-ui-dm"); err != nil {
		return nil, err
	}
	if err := d.Open(ctx, baseURL+"/groups/"+groupID); err != nil {
		return nil, err
	}
	if err := check(c.Selectors.GroupID, ""); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.GroupID, groupID); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.GroupMessage, "human-ui-group"); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.GroupSend); err != nil {
		return nil, err
	}
	if err := check(c.Selectors.GroupVisible, "human-ui-group"); err != nil {
		return nil, err
	}
	if c.Selectors.OrderVisible != "" {
		if err := check(c.Selectors.OrderVisible, "human-ui-group"); err != nil {
			return nil, err
		}
	}
	if c.Selectors.NavForums != "" {
		if err := d.Click(ctx, c.Selectors.NavForums); err != nil {
			return nil, err
		}
		if err := check(c.Selectors.PostTitle, ""); err != nil {
			return nil, err
		}
	}
	if err := d.Fill(ctx, c.Selectors.PostTitle, "human-ui-post"); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.PostContent, "human-ui-post-content"); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.PostCreate); err != nil {
		return nil, err
	}
	if err := check(c.Selectors.PostVisible, "human-ui-post-content"); err != nil {
		return nil, err
	}
	if err := d.Open(ctx, baseURL+"/posts/"+postID); err != nil {
		return nil, err
	}
	if err := check(c.Selectors.ThreadID, ""); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.ThreadID, postID); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.CommentContent, "human-ui-comment"); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.CommentSend); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.Follow); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.React); err != nil {
		return nil, err
	}
	for _, p := range []struct{ s, n string }{{c.Selectors.CommentVisible, "human-ui-comment"}, {c.Selectors.ThreadVisible, postID}, {c.Selectors.ThreadStructure, "human-ui-comment"}} {
		if p.s != "" {
			if err := check(p.s, p.n); err != nil {
				return nil, err
			}
		}
	}
	if c.Selectors.NavInbox != "" && c.Selectors.Notification != "" {
		if err := d.Open(ctx, baseURL+"/notifications"); err != nil {
			return nil, err
		}
		if err := check(c.Selectors.Notification, "notification"); err != nil {
			return nil, err
		}
	}
	s, e := d.Snapshot(ctx)
	if e != nil {
		return nil, e
	}
	return []string{s}, nil
}

func waitVisible(ctx context.Context, d Driver, selector, needle string) error {
	deadline := time.Now().Add(5 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		if err := d.AssertVisible(ctx, selector, needle); err == nil {
			return nil
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return last
}

// RunAdminFlow checks the human administration surface using only visible
// controls. Empty optional action selectors are skipped so the same driver can
// target an implementation that exposes a read-only subset of administration.
func RunAdminFlow(ctx context.Context, d Driver, c Config, serverID string) ([]string, error) {
	d = observedDriver{Driver: d, observer: c.Observer}
	if err := d.Open(ctx, c.URL); err != nil {
		return nil, err
	}
	defer d.Close(context.Background())
	if c.Selectors.IdentityID != "" && c.Selectors.IdentityLoad != "" {
		if err := waitVisible(ctx, d, c.Selectors.IdentityID, ""); err != nil {
			return nil, err
		}
		if err := d.Fill(ctx, c.Selectors.IdentityID, c.IdentityID); err != nil {
			return nil, err
		}
		if err := d.Click(ctx, c.Selectors.IdentityLoad); err != nil {
			return nil, err
		}
		if c.Selectors.IdentityVisible != "" {
			if err := waitVisible(ctx, d, c.Selectors.IdentityVisible, "Connected"); err != nil {
				return nil, err
			}
		}
	}
	if c.Selectors.NavAdmin != "" {
		if err := d.Open(ctx, strings.TrimRight(c.URL, "/")+"/settings"); err != nil {
			return nil, err
		}
		if c.Selectors.ServerID != "" {
			if err := waitVisible(ctx, d, c.Selectors.ServerID, ""); err != nil {
				return nil, err
			}
		}
	}
	if c.Selectors.ServerID != "" {
		if err := d.Fill(ctx, c.Selectors.ServerID, serverID); err != nil {
			return nil, err
		}
		if err := d.Press(ctx, c.Selectors.ServerID, "Enter"); err != nil {
			return nil, err
		}
	}
	for _, p := range []struct{ selector, needle string }{
		{c.Selectors.ServerVisible, serverID},
		{c.Selectors.MemberVisible, "member"},
		{c.Selectors.RequestVisible, "request"},
		{c.Selectors.GroupAdmin, "group"},
		{c.Selectors.Moderation, "post"},
		{c.Selectors.AuditVisible, "audit"},
	} {
		if p.selector != "" {
			if err := waitVisible(ctx, d, p.selector, p.needle); err != nil {
				return nil, err
			}
		}
	}
	if c.Selectors.ApproveRequest != "" {
		if err := d.Click(ctx, c.Selectors.ApproveRequest); err != nil {
			return nil, err
		}
	}
	if c.Selectors.RoleParticipant != "" && c.Selectors.RoleValue != "" && c.Selectors.RoleSave != "" {
		if err := d.Fill(ctx, c.Selectors.RoleParticipant, c.RoleParticipant); err != nil {
			return nil, err
		}
		if err := d.Fill(ctx, c.Selectors.RoleValue, "moderator"); err != nil {
			return nil, err
		}
		if err := d.Click(ctx, c.Selectors.RoleSave); err != nil {
			return nil, err
		}
	}
	s, err := d.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return []string{s}, nil
}
