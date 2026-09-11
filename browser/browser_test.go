package browser

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fakeDriver struct {
	calls       []string
	texts       map[string]string
	routes      map[string]string
	currentPath string
}

func (f *fakeDriver) record(v string)                    { f.calls = append(f.calls, v) }
func (f *fakeDriver) Open(context.Context, string) error { f.record("open"); return nil }
func (f *fakeDriver) Snapshot(context.Context) (string, error) {
	f.record("snapshot")
	return "navigation main heading snapshot", nil
}
func (f *fakeDriver) URL(context.Context) (string, error) {
	f.record("url")
	return "http://ui" + f.currentPath, nil
}
func (f *fakeDriver) SetViewport(_ context.Context, width, height int) error {
	f.record(fmt.Sprintf("viewport:%dx%d", width, height))
	return nil
}
func (f *fakeDriver) Click(_ context.Context, selector string) error {
	f.record("click:" + selector)
	if path, ok := f.routes[selector]; ok {
		f.currentPath = path
	}
	return nil
}
func (f *fakeDriver) Fill(_ context.Context, selector, value string) error {
	f.record("fill:" + selector + "=" + value)
	return nil
}
func (f *fakeDriver) Press(_ context.Context, selector, key string) error {
	f.record("press:" + selector + "=" + key)
	return nil
}
func (f *fakeDriver) Text(_ context.Context, selector string) (string, error) {
	f.record("text:" + selector)
	return f.texts[selector], nil
}
func (f *fakeDriver) AssertVisible(ctx context.Context, selector, needle string) error {
	value, _ := f.Text(ctx, selector)
	if !strings.Contains(value, needle) {
		return &assertionError{selector: selector, needle: needle}
	}
	return nil
}
func (f *fakeDriver) Close(context.Context) error { f.record("close"); return nil }

type assertionError struct{ selector, needle string }

func (e *assertionError) Error() string { return "missing " + e.needle + " in " + e.selector }

func TestRunAdminFlowUsesVisibleAssertionsAndActions(t *testing.T) {
	f := &fakeDriver{texts: map[string]string{
		"server": "server-1", "members": "member agent-a", "requests": "request pending", "groups": "group", "moderation": "post", "audit": "audit entry",
	}}
	c := Config{URL: "http://ui", IdentityID: "human", RoleParticipant: "agent-a", Selectors: Selectors{IdentityID: "identity", IdentityLoad: "connect", ServerID: "server-input", ServerVisible: "server", MemberVisible: "members", RequestVisible: "requests", ApproveRequest: "approve", RoleParticipant: "role-participant", RoleValue: "role-value", RoleSave: "role-save", GroupAdmin: "groups", Moderation: "moderation", AuditVisible: "audit"}}
	if _, err := RunAdminFlow(context.Background(), f, c, "server-1"); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(f.calls, "|")
	for _, want := range []string{"fill:identity=human", "click:connect", "fill:server-input=server-1", "click:approve", "fill:role-participant=agent-a", "fill:role-value=moderator", "text:audit"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("calls omitted %q: %s", want, joined)
		}
	}
}

func TestVerifyApplicationIARequiresLandmarksRoutesAndMobileSnapshot(t *testing.T) {
	f := &fakeDriver{routes: map[string]string{"inbox": "/inbox", "servers": "/servers", "members": "/members", "groups": "/groups", "forums": "/forums", "admin": "/servers/server-1/settings"}}
	selectors := Selectors{NavInbox: "inbox", NavServers: "servers", NavMembers: "members", NavGroups: "groups", NavForums: "forums", NavAdmin: "admin"}
	if err := verifyApplicationIA(context.Background(), f, "http://ui/", selectors); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(f.calls, "|")
	for _, want := range []string{"click:servers", "url", "viewport:390x844", "snapshot", "viewport:1440x900", "open"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("calls omitted %q: %s", want, joined)
		}
	}
}

type eventuallyVisibleDriver struct {
	fakeDriver
	attempts int
}

func (f *eventuallyVisibleDriver) AssertVisible(_ context.Context, selector, needle string) error {
	f.attempts++
	if f.attempts < 3 {
		return &assertionError{selector: selector, needle: needle}
	}
	return nil
}

func TestWaitVisibleRetriesAsyncUIState(t *testing.T) {
	f := &eventuallyVisibleDriver{}
	if err := waitVisible(context.Background(), f, "presence", "online"); err != nil {
		t.Fatal(err)
	}
	if f.attempts != 3 {
		t.Fatalf("attempts = %d, want 3", f.attempts)
	}
}
