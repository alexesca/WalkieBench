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
	if !strings.Contains(v, needle) {
		return fmt.Errorf("selector %q does not contain %q; got %q", s, needle, v)
	}
	return nil
}
func (a AgentBrowser) Close(ctx context.Context) error { _, e := a.run(ctx, "close"); return e }

type Selectors struct {
	IdentityID     string `json:"identity_id"`
	IdentityLoad   string `json:"identity_load"`
	DMRecipient    string `json:"dm_recipient"`
	DMContent      string `json:"dm_content"`
	DMSend         string `json:"dm_send"`
	DMVisible      string `json:"dm_visible"`
	GroupName      string `json:"group_name"`
	GroupCreate    string `json:"group_create"`
	GroupID        string `json:"group_id"`
	GroupMessage   string `json:"group_message"`
	GroupSend      string `json:"group_send"`
	GroupVisible   string `json:"group_visible"`
	PostTitle      string `json:"post_title"`
	PostContent    string `json:"post_content"`
	PostCreate     string `json:"post_create"`
	PostVisible    string `json:"post_visible"`
	ThreadID       string `json:"thread_id"`
	CommentContent string `json:"comment_content"`
	CommentSend    string `json:"comment_send"`
	CommentVisible string `json:"comment_visible"`
	Follow         string `json:"follow"`
	React          string `json:"react"`
	Presence       string `json:"presence"`
}
type Config struct {
	URL        string
	Selectors  Selectors
	IdentityID string
	Observer   func(string, time.Duration, error)
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

// RunHumanFlow performs all human-visible major actions. Selectors use
// data-testid values by default; a future UI may supply equivalent refs.
func RunHumanFlow(ctx context.Context, d Driver, c Config, groupID, postID, agentID string) ([]string, error) {
	d = observedDriver{Driver: d, observer: c.Observer}
	if err := d.Open(ctx, c.URL); err != nil {
		return nil, err
	}
	defer d.Close(context.Background())
	if c.Selectors.IdentityID != "" && c.Selectors.IdentityLoad != "" {
		if err := d.Fill(ctx, c.Selectors.IdentityID, c.IdentityID); err != nil {
			return nil, err
		}
		if err := d.Click(ctx, c.Selectors.IdentityLoad); err != nil {
			return nil, err
		}
	}
	check := func(sel, needle string) error { return d.AssertVisible(ctx, sel, needle) }
	if err := d.Fill(ctx, c.Selectors.DMRecipient, agentID); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.DMContent, "human-ui-dm"); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.DMSend); err != nil {
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
	if err := d.Fill(ctx, c.Selectors.PostTitle, "human-ui-post"); err != nil {
		return nil, err
	}
	if err := d.Fill(ctx, c.Selectors.PostContent, "human-ui-post-content"); err != nil {
		return nil, err
	}
	if err := d.Click(ctx, c.Selectors.PostCreate); err != nil {
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
	for _, p := range []struct{ s, n string }{{c.Selectors.DMVisible, "human-ui-dm"}, {c.Selectors.GroupVisible, "human-ui-group"}, {c.Selectors.PostVisible, "human-ui-post-content"}, {c.Selectors.CommentVisible, "human-ui-comment"}, {c.Selectors.Presence, "online"}} {
		if p.s != "" {
			if err := check(p.s, p.n); err != nil {
				return nil, err
			}
		}
	}
	s, e := d.Snapshot(ctx)
	if e != nil {
		return nil, e
	}
	return []string{s}, nil
}
