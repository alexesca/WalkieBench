package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"walkiebench/benchmark"
	"walkiebench/browser"
	"walkiebench/contract"
	"walkiebench/telemetry"
	"walkiebench/transport"
)

type jsonFactory struct {
	endpoint   string
	httpClient *http.Client
	collector  *telemetry.Collector
}

func (f jsonFactory) New(ctx context.Context, id *contract.Identity) (contract.Client, error) {
	token := ""
	if id != nil {
		token = id.SessionToken
	}
	return transport.New(transport.Config{Endpoint: f.endpoint, HTTPClient: f.httpClient, BearerToken: token, Observer: func(e transport.Event) {
		plaintext := f.collector.ContainsPlaintext(e.RequestBody) || f.collector.ContainsPlaintext(e.ResponseBody)
		f.collector.ObserveWire(e.Operation, e.RequestSize, e.ResponseSize, e.Latency, e.Err, plaintext)
	}}), nil
}

func (f jsonFactory) NewTransport(ctx context.Context, name string, id *contract.Identity) (contract.Client, error) {
	if name != "jsonrpc" {
		return nil, fmt.Errorf("transport %q is not configured for this benchmark endpoint", name)
	}
	return f.New(ctx, id)
}

func main() {
	endpoint := flag.String("endpoint", "", "JSON-RPC endpoint implementing the WalkieBench contract")
	uiURL := flag.String("ui-url", "", "human UI URL; required for the browser gate")
	browserCommand := flag.String("browser-command", "agent-browser", "Chromium driver command")
	output := flag.String("output", "walkiebench-scorecard.json", "scorecard JSON output path")
	workers := flag.Int("workers", 3, "concurrent writers and disconnected load multiplier")
	historyMessages := flag.Int("history-messages", 2000, "messages in growing-history scenario")
	historyComments := flag.Int("history-comments", 1000, "comments in growing-history scenario")
	resourceCommand := flag.String("resource-command", "", "optional command returning telemetry.Resource JSON for server resource samples")
	selectorsFile := flag.String("browser-selectors", "", "JSON file containing browser selector overrides")
	profile := flag.String("profile", "full", "benchmark profile (use --help with profiles documented in README)")
	seed := flag.Int64("seed", 20260910, "deterministic scenario seed")
	requireV2 := flag.Bool("require-v2", false, "treat unavailable V2 capabilities as invalid even outside the full profile")
	flag.Parse()
	if *endpoint == "" {
		fmt.Fprintln(os.Stderr, "--endpoint is required")
		os.Exit(2)
	}
	t := telemetry.New()
	selectors := defaultSelectors()
	if *selectorsFile != "" {
		b, e := os.ReadFile(*selectorsFile)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(2)
		}
		if e = json.Unmarshal(b, &selectors); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(2)
		}
	}
	var driver browser.Driver
	if *uiURL != "" {
		driver = browser.AgentBrowser{Command: *browserCommand}
	}
	cfg := benchmark.Config{Workers: *workers, HistoryMessages: *historyMessages, HistoryComments: *historyComments, ResourceCommand: *resourceCommand, BrowserURL: *uiURL, BrowserCommand: *browserCommand, BrowserSelectors: selectors, Profile: *profile, Seed: *seed, RequireV2: *requireV2}
	factory := jsonFactory{endpoint: *endpoint, httpClient: &http.Client{Timeout: 30 * time.Second}, collector: t}
	score := benchmark.New(factory, cfg, t, driver).Run(context.Background())
	if e := t.WriteJSON(*output); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	b, _ := json.MarshalIndent(score, "", "  ")
	fmt.Println(string(b))
	if !score.Passed {
		os.Exit(1)
	}
}

func defaultSelectors() browser.Selectors {
	return browser.Selectors{IdentityID: `[data-testid="identity-id"]`, IdentityLoad: `[data-testid="identity-load"]`, DMRecipient: `[data-testid="dm-recipient"]`, DMContent: `[data-testid="dm-content"]`, DMSend: `[data-testid="dm-send"]`, DMVisible: `[data-testid="dm-messages"]`, GroupID: `[data-testid="group-id"]`, GroupMessage: `[data-testid="group-message"]`, GroupSend: `[data-testid="group-send"]`, GroupVisible: `[data-testid="group-messages"]`, PostTitle: `[data-testid="post-title"]`, PostContent: `[data-testid="post-content"]`, PostCreate: `[data-testid="post-create"]`, PostVisible: `[data-testid="posts"]`, ThreadID: `[data-testid="thread-id"]`, CommentContent: `[data-testid="comment-content"]`, CommentSend: `[data-testid="comment-send"]`, CommentVisible: `[data-testid="comments"]`, Follow: `[data-testid="thread-follow"]`, React: `[data-testid="react"]`, Presence: `[data-testid="presence"]`}
}
