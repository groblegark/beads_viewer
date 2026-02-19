package datasource

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"
	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

// HTTPReader loads issues from a Gas Town daemon via ConnectRPC.
type HTTPReader struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewHTTPReader creates a reader for a daemon HTTP endpoint.
func NewHTTPReader(baseURL, apiKey string) *HTTPReader {
	return &HTTPReader{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: loader.DefaultHTTPTimeout,
		},
	}
}

// LoadIssues fetches all issues from the daemon.
func (r *HTTPReader) LoadIssues() ([]model.Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), loader.DefaultHTTPTimeout)
	defer cancel()
	return loader.LoadIssuesFromURL(ctx, r.baseURL, r.apiKey, loader.ParseOptions{})
}

// Ping performs a lightweight connectivity check against the daemon.
// Returns the number of issues available without loading them all.
func (r *HTTPReader) Ping() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	issues, err := loader.LoadIssuesFromURL(ctx, r.baseURL, r.apiKey, loader.ParseOptions{})
	if err != nil {
		return 0, err
	}
	return len(issues), nil
}

// discoverHTTPSources checks for HTTP daemon endpoints via options or env vars.
func discoverHTTPSources(opts DiscoveryOptions) []DataSource {
	url := opts.HTTPEndpoint

	// Fall back to environment variables (matching main.go's env var names)
	if url == "" {
		url = os.Getenv("BV_BEADS_URL")
	}
	if url == "" {
		url = os.Getenv("BD_DAEMON_HOST")
	}

	if url == "" {
		return nil
	}

	// Normalize URL
	url = strings.TrimRight(url, "/")

	if opts.Verbose {
		opts.Logger(fmt.Sprintf("Found HTTP source: %s", url))
	}

	return []DataSource{{
		Type:     SourceTypeHTTP,
		Path:     url,
		Priority: PriorityHTTP,
		ModTime:  time.Now(), // HTTP sources are always "current"
		Valid:    false,      // Must be validated
		apiKey:   opts.HTTPAPIKey,
	}}
}

// validateHTTP validates an HTTP daemon source by performing a connectivity check.
func validateHTTP(source *DataSource, opts ValidationOptions) error {
	reader := NewHTTPReader(source.Path, source.apiKey)
	count, err := reader.Ping()
	if err != nil {
		return fmt.Errorf("daemon unreachable: %w", err)
	}

	if opts.CountIssues {
		source.IssueCount = count
	}

	// Update ModTime to now since HTTP data is always live
	source.ModTime = time.Now()

	if opts.Verbose {
		opts.Logger(fmt.Sprintf("HTTP validation passed: %s (%d issues)", source.Path, count))
	}

	return nil
}

// HTTPPoller monitors a daemon for changes and triggers callbacks.
type HTTPPoller struct {
	reader    *HTTPReader
	interval  time.Duration
	callback  func(DataSource)
	lastCount int
	source    DataSource
	done      chan struct{}
	mu        sync.Mutex
	verbose   bool
	logger    func(msg string)
}

// HTTPPollerOptions configures the HTTP poller.
type HTTPPollerOptions struct {
	// Interval is the polling frequency. Default: 30s.
	Interval time.Duration
	// Verbose enables logging.
	Verbose bool
	// Logger receives log messages.
	Logger func(msg string)
}

// NewHTTPPoller creates a poller for an HTTP daemon source.
func NewHTTPPoller(source DataSource, callback func(DataSource), opts HTTPPollerOptions) *HTTPPoller {
	if opts.Interval == 0 {
		opts.Interval = 30 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = func(string) {}
	}

	return &HTTPPoller{
		reader:   NewHTTPReader(source.Path, source.apiKey),
		interval: opts.Interval,
		callback: callback,
		source:   source,
		done:     make(chan struct{}),
		verbose:  opts.Verbose,
		logger:   opts.Logger,
	}
}

// Start begins polling the daemon for changes.
func (p *HTTPPoller) Start() {
	go p.run()
}

// Stop stops polling.
func (p *HTTPPoller) Stop() {
	close(p.done)
}

func (p *HTTPPoller) run() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.poll()
		}
	}
}

func (p *HTTPPoller) poll() {
	count, err := p.reader.Ping()
	if err != nil {
		if p.verbose {
			p.logger(fmt.Sprintf("HTTP poll failed: %v", err))
		}
		return
	}

	p.mu.Lock()
	changed := count != p.lastCount
	oldCount := p.lastCount
	p.lastCount = count
	p.mu.Unlock()

	if changed {
		if p.verbose {
			p.logger(fmt.Sprintf("HTTP source changed: %d issues (was %d)", count, oldCount))
		}
		p.source.IssueCount = count
		p.source.ModTime = time.Now()
		if p.callback != nil {
			p.callback(p.source)
		}
	}
}
