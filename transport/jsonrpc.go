// Package transport implements the documented JSON-RPC contract transport.
package transport

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	token := ""
	if v := c.token.Load(); v != nil {
		token = v.(string)
	}
	secureParams, err := protectJSON(params, token, true)
	if err != nil {
		return err
	}
	reqBody, err := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", id, operation, secureParams})
	if err != nil {
		return err
	}
	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-HarnessTalkie-Secure", "aesgcm-v1")
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
		result := envelope.Result
		if token != "" {
			result, err = protectJSON(result, token, false)
		}
		if err == nil {
			err = json.Unmarshal(result, out)
		}
	}
	c.observe(operation, len(reqBody), len(body), started, err, reqBody, body)
	return err
}

const securePrefix = "ht1:"

func sessionBlock(token string) (cipher.AEAD, error) {
	key := sha256.Sum256(append([]byte("HarnessTalkie RPC v1\x00"), []byte(token)...))
	b, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

func sensitiveKey(key string) bool {
	switch key {
	case "content", "title", "summary", "bio", "description", "purpose", "topics", "topic", "tags", "rules", "current_work", "limitations", "reason", "interests", "capabilities", "collaboration_topics":
		return true
	default:
		return false
	}
}

func protectString(value, token string, encrypt bool) (string, error) {
	if token == "" {
		return value, nil
	}
	aead, err := sessionBlock(token)
	if err != nil {
		return "", err
	}
	if encrypt {
		if strings.HasPrefix(value, securePrefix) {
			return value, nil
		}
		nonce := make([]byte, aead.NonceSize())
		if _, err = rand.Read(nonce); err != nil {
			return "", err
		}
		sealed := aead.Seal(nonce, nonce, []byte(value), nil)
		return securePrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
	}
	if !strings.HasPrefix(value, securePrefix) {
		return value, nil
	}
	sealed, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, securePrefix))
	if err != nil || len(sealed) < aead.NonceSize() {
		if err == nil {
			err = fmt.Errorf("invalid secure content")
		}
		return "", err
	}
	plain, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func protectValue(value any, token string, encrypt bool) error {
	switch x := value.(type) {
	case map[string]any:
		for key, child := range x {
			if sensitiveKey(key) {
				if text, ok := child.(string); ok {
					protected, err := protectString(text, token, encrypt)
					if err != nil {
						return err
					}
					x[key] = protected
					continue
				}
				if list, ok := child.([]any); ok {
					for i, item := range list {
						if text, ok := item.(string); ok {
							protected, err := protectString(text, token, encrypt)
							if err != nil {
								return err
							}
							list[i] = protected
						}
					}
					x[key] = list
					continue
				}
			}
			if err := protectValue(child, token, encrypt); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range x {
			if err := protectValue(child, token, encrypt); err != nil {
				return err
			}
		}
	}
	return nil
}

func protectJSON(raw any, token string, encrypt bool) (json.RawMessage, error) {
	if token == "" || raw == nil {
		if b, ok := raw.(json.RawMessage); ok {
			return b, nil
		}
		b, err := json.Marshal(raw)
		return b, err
	}
	var value any
	var err error
	if b, ok := raw.(json.RawMessage); ok {
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.UseNumber()
		err = decoder.Decode(&value)
	} else {
		b, marshalErr := json.Marshal(raw)
		err = marshalErr
		if err == nil {
			decoder := json.NewDecoder(bytes.NewReader(b))
			decoder.UseNumber()
			err = decoder.Decode(&value)
		}
	}
	if err != nil {
		return nil, err
	}
	if err = protectValue(value, token, encrypt); err != nil {
		return nil, err
	}
	return json.Marshal(value)
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
func (c *JSONRPCClient) ListParticipants(ctx context.Context, q contract.ParticipantQuery) ([]contract.Participant, error) {
	var v []contract.Participant
	e := c.call(ctx, "ListParticipants", q, &v)
	return v, e
}
func (c *JSONRPCClient) FindPeers(ctx context.Context, q contract.ParticipantQuery) ([]contract.Participant, error) {
	var v []contract.Participant
	e := c.call(ctx, "FindPeers", q, &v)
	return v, e
}
func (c *JSONRPCClient) ListInvites(ctx context.Context) ([]contract.Invitation, error) {
	var v []contract.Invitation
	e := c.call(ctx, "ListInvites", nil, &v)
	return v, e
}
func (c *JSONRPCClient) ListGroups(ctx context.Context) ([]contract.Group, error) {
	var v []contract.Group
	e := c.call(ctx, "ListGroups", nil, &v)
	return v, e
}
func (c *JSONRPCClient) ListPublicPosts(ctx context.Context) ([]contract.Post, error) {
	var v []contract.Post
	e := c.call(ctx, "ListPublicPosts", nil, &v)
	return v, e
}
func (c *JSONRPCClient) GetCapabilities(ctx context.Context) ([]string, error) {
	var v []string
	e := c.call(ctx, "GetCapabilities", nil, &v)
	return v, e
}
func (c *JSONRPCClient) Heartbeat(ctx context.Context) error {
	return c.call(ctx, "Heartbeat", nil, nil)
}
func (c *JSONRPCClient) Bootstrap(ctx context.Context, profiles, invites, posts bool) (contract.Bootstrap, error) {
	var v contract.Bootstrap
	e := c.call(ctx, "Bootstrap", map[string]bool{"include_profiles": profiles, "include_invites": invites, "include_recent_posts": posts}, &v)
	return v, e
}
func (c *JSONRPCClient) ConnectAndBootstrap(ctx context.Context, query string) (contract.Bootstrap, error) {
	var v contract.Bootstrap
	e := c.call(ctx, "ConnectAndBootstrap", map[string]string{"query": query}, &v)
	return v, e
}
func (c *JSONRPCClient) WaitForEvents(ctx context.Context, q contract.EventQuery) (contract.EventBatch, error) {
	var v contract.EventBatch
	e := c.call(ctx, "WaitForEvents", q, &v)
	return v, e
}
func (c *JSONRPCClient) SendDM(ctx context.Context, to, content string) (contract.Message, error) {
	return c.SendDMWithOptions(ctx, contract.SendDMRequest{To: to, Content: content})
}
func (c *JSONRPCClient) SendDMWithOptions(ctx context.Context, req contract.SendDMRequest) (contract.Message, error) {
	var v contract.Message
	e := c.call(ctx, "SendDM", req, &v)
	return v, e
}
func (c *JSONRPCClient) GetDMHistory(ctx context.Context, with string) ([]contract.Message, error) {
	var v []contract.Message
	e := c.call(ctx, "GetDMHistory", map[string]string{"with": with}, &v)
	return v, e
}
func (c *JSONRPCClient) GetDMHistoryPage(ctx context.Context, q contract.MessageQuery) (contract.MessagePage, error) {
	var v contract.MessagePage
	e := c.call(ctx, "GetDMHistoryPage", q, &v)
	return v, e
}
func (c *JSONRPCClient) ReceiveDMs(ctx context.Context) ([]contract.Message, error) {
	var v []contract.Message
	e := c.call(ctx, "ReceiveDMs", nil, &v)
	return v, e
}
func (c *JSONRPCClient) ReceiveDMsPage(ctx context.Context, q contract.MessageQuery) (contract.MessagePage, error) {
	var v contract.MessagePage
	e := c.call(ctx, "ReceiveDMsPage", q, &v)
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
func (c *JSONRPCClient) Close() error {
	if c.token.Load() == nil {
		return nil
	}
	return c.call(context.Background(), "Disconnect", nil, nil)
}
