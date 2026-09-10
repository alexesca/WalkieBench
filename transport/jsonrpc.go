// Package transport implements the documented JSON-RPC contract transport.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"walkiebench/contract"
)

type Event struct {
	Operation    string
	RequestSize  int64
	ResponseSize int64
	Latency      time.Duration
	Err          error
	RequestBody  []byte
	ResponseBody []byte
}

type Config struct {
	Endpoint    string
	HTTPClient  *http.Client
	BearerToken string
	Observer    func(Event)
}

type JSONRPCClient struct {
	cfg    Config
	nextID uint64
	token  atomic.Value
}

func New(cfg Config) *JSONRPCClient {
	c := &JSONRPCClient{cfg: cfg}
	if cfg.HTTPClient == nil {
		c.cfg.HTTPClient = http.DefaultClient
	}
	if cfg.BearerToken != "" {
		c.token.Store(cfg.BearerToken)
	}
	return c
}

func (c *JSONRPCClient) call(ctx context.Context, operation string, params any, out any) error {
	id := atomic.AddUint64(&c.nextID, 1)
	reqBody, err := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", id, operation, params})
	if err != nil {
		return err
	}
	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if v := c.token.Load(); v != nil {
		req.Header.Set("Authorization", "Bearer "+v.(string))
	}
	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		c.observe(operation, len(reqBody), 0, started, err, reqBody, nil)
		return err
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		err = readErr
	} else if closeErr != nil {
		err = closeErr
	}
	if resp.StatusCode >= 400 && err == nil {
		err = fmt.Errorf("http status %s", resp.Status)
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err == nil {
		err = json.Unmarshal(body, &envelope)
	}
	if err == nil && envelope.Error != nil {
		err = fmt.Errorf("contract error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if err == nil && out != nil {
		err = json.Unmarshal(envelope.Result, out)
	}
	c.observe(operation, len(reqBody), len(body), started, err, reqBody, body)
	return err
}

func (c *JSONRPCClient) observe(op string, request, response int, started time.Time, err error, body, responseBody []byte) {
	if c.cfg.Observer != nil {
		c.cfg.Observer(Event{Operation: op, RequestSize: int64(request), ResponseSize: int64(response), Latency: time.Since(started), Err: err, RequestBody: body, ResponseBody: responseBody})
	}
}

func (c *JSONRPCClient) CreateOrLoadIdentity(ctx context.Context, name string) (contract.Identity, error) {
	var v contract.Identity
	e := c.call(ctx, "CreateOrLoadIdentity", map[string]string{"identity": name}, &v)
	if v.SessionToken != "" {
		c.token.Store(v.SessionToken)
	}
	return v, e
}
func (c *JSONRPCClient) PublishProfile(ctx context.Context, p contract.Profile) error {
	return c.call(ctx, "PublishProfile", p, nil)
}
func (c *JSONRPCClient) GetPresence(ctx context.Context, id string) (contract.Presence, error) {
	var v contract.Presence
	e := c.call(ctx, "GetPresence", map[string]string{"identity_id": id}, &v)
	return v, e
}
func (c *JSONRPCClient) ListOnline(ctx context.Context) ([]contract.Presence, error) {
	var v []contract.Presence
	e := c.call(ctx, "ListOnline", nil, &v)
	return v, e
}
func (c *JSONRPCClient) ConnectTo(ctx context.Context, id string) error {
	return c.call(ctx, "ConnectTo", map[string]string{"identity_id": id}, nil)
}
func (c *JSONRPCClient) ListContacts(ctx context.Context) ([]contract.Contact, error) {
	var v []contract.Contact
	e := c.call(ctx, "ListContacts", nil, &v)
	return v, e
}
func (c *JSONRPCClient) SendDM(ctx context.Context, to, content string) (contract.Message, error) {
	var v contract.Message
	e := c.call(ctx, "SendDM", map[string]string{"to": to, "content": content}, &v)
	return v, e
}
func (c *JSONRPCClient) GetDMHistory(ctx context.Context, with string) ([]contract.Message, error) {
	var v []contract.Message
	e := c.call(ctx, "GetDMHistory", map[string]string{"with": with}, &v)
	return v, e
}
func (c *JSONRPCClient) ReceiveDMs(ctx context.Context) ([]contract.Message, error) {
	var v []contract.Message
	e := c.call(ctx, "ReceiveDMs", nil, &v)
	return v, e
}
func (c *JSONRPCClient) MarkRead(ctx context.Context, ids []string) error {
	return c.call(ctx, "MarkRead", map[string]any{"message_ids": ids}, nil)
}
func (c *JSONRPCClient) CreateGroup(ctx context.Context, n string) (contract.Group, error) {
	var v contract.Group
	e := c.call(ctx, "CreateGroup", map[string]string{"name": n}, &v)
	return v, e
}
func (c *JSONRPCClient) Invite(ctx context.Context, g, u string) error {
	return c.call(ctx, "Invite", map[string]string{"group": g, "user": u}, nil)
}
func (c *JSONRPCClient) Join(ctx context.Context, g string) error {
	return c.call(ctx, "Join", map[string]string{"group": g}, nil)
}
func (c *JSONRPCClient) Leave(ctx context.Context, g string) error {
	return c.call(ctx, "Leave", map[string]string{"group": g}, nil)
}
func (c *JSONRPCClient) SendGroupMessage(ctx context.Context, g, m string) (contract.Message, error) {
	var v contract.Message
	e := c.call(ctx, "SendGroupMessage", map[string]string{"group": g, "content": m}, &v)
	return v, e
}
func (c *JSONRPCClient) GetGroupHistory(ctx context.Context, g string) ([]contract.Message, error) {
	var v []contract.Message
	e := c.call(ctx, "GetGroupHistory", map[string]string{"group": g}, &v)
	return v, e
}
func (c *JSONRPCClient) CreatePost(ctx context.Context, t, b string) (contract.Post, error) {
	var v contract.Post
	e := c.call(ctx, "CreatePost", map[string]string{"title": t, "content": b}, &v)
	return v, e
}
func (c *JSONRPCClient) Comment(ctx context.Context, p, b string) (contract.CommentNode, error) {
	var v contract.CommentNode
	e := c.call(ctx, "Comment", map[string]string{"post_or_comment": p, "content": b}, &v)
	return v, e
}
func (c *JSONRPCClient) GetThread(ctx context.Context, p string) (contract.Thread, error) {
	var v contract.Thread
	e := c.call(ctx, "GetThread", map[string]string{"post": p}, &v)
	return v, e
}
func (c *JSONRPCClient) FollowThread(ctx context.Context, p string) error {
	return c.call(ctx, "FollowThread", map[string]string{"post": p}, nil)
}
func (c *JSONRPCClient) UnfollowThread(ctx context.Context, p string) error {
	return c.call(ctx, "UnfollowThread", map[string]string{"post": p}, nil)
}
func (c *JSONRPCClient) React(ctx context.Context, target, reaction string) error {
	return c.call(ctx, "React", map[string]string{"target": target, "reaction": reaction}, nil)
}
func (c *JSONRPCClient) Resume(ctx context.Context) (contract.ResumeResult, error) {
	var v contract.ResumeResult
	e := c.call(ctx, "Resume", nil, &v)
	return v, e
}
func (c *JSONRPCClient) Close() error { return nil }
