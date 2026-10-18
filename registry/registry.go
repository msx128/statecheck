package registry

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// urls would be added with web protocol

type Target struct {
	ID, URL                  string
	isPolling                bool
	ctx                      context.Context
	cancelTarget, cancelPoll context.CancelFunc
}

func NewTarget(ctx context.Context, url string) *Target {
	newCtx, cancel := context.WithCancel(ctx)
	return &Target{ID: uuid.NewString(), URL: url,
		isPolling: false, ctx: newCtx,
		cancelTarget: cancel, cancelPoll: nil}
}

type Registry struct {
	client  *http.Client
	targets map[string]*Target
	root    context.Context
	mu      sync.Mutex
}

func NewRegistry(ctx context.Context, client *http.Client) *Registry {
	return &Registry{client: client,
		targets: make(map[string]*Target),
		root:    ctx, mu: sync.Mutex{}}
}

func (r *Registry) Add(tag string, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, some := r.targets[tag]; some {
		return fmt.Errorf("entry with this tag: %s already exists", tag)
	}
	target := NewTarget(r.root, url)
	r.targets[tag] = target
	return nil
}

func (r *Registry) Delete(tag string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	target, some := r.targets[tag]
	if !some {
		return fmt.Errorf("entry with this tag: %s does not exists", tag)
	}
	target.cancelTarget()
	delete(r.targets, tag)
	return nil
}

func (r *Registry) Poll(tag string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	target, ok := r.targets[tag]
	if !ok {
		return fmt.Errorf("failed to poll target, not exists")
	}
	if target.isPolling {
		return fmt.Errorf("target is already polling")
	}
	target.isPolling = true // might be source of bug in some cases
	ctx, cancel := context.WithCancel(target.ctx)
	target.cancelPoll = cancel
	go func() {
		pollHelper(target, r)
		// make this configurable in a future
		// maybe add ticker field in target
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				if target.ctx.Err() != nil {
					log.Printf("poll stopped by shutdown or deleting target")
				} else {
					log.Printf("poll canceled")
				}
				return

			case <-ticker.C:
				pollHelper(target, r)
			}
		}
	}()
	return nil
}

func pollHelper(target *Target, r *Registry) {
	body, err := getData(target.ctx, target.URL, r.client)
	if err != nil {
		log.Printf("error while polling: %v", err)
	} else {
		fmt.Printf("ctx: %v, get: %s", target.ctx, body)
	}
}

func (r *Registry) StopPolling(tag string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	target, ok := r.targets[tag]
	if !ok {
		return fmt.Errorf("failed to stop polling target, not exists")
	}
	if !target.isPolling {
		return fmt.Errorf("target is not polling")
	}
	target.cancelPoll()
	target.isPolling = false
	return nil
}

func getData(ctx context.Context, url string, client *http.Client) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponseCode(resp.StatusCode); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return body, nil
}

// may do custom struct or enum
func checkResponseCode(code int) error {
	bad := code < 200 || code > 299
	if bad {
		return fmt.Errorf("get bad status code: %d", code)
	}
	return nil
}
