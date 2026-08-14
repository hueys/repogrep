// Package github implements source.Source by pulling a user's starred
// repos (and their READMEs) from the GitHub REST API. It never clones
// repos — only metadata + README content are fetched.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/go-github/v75/github"
	"github.com/hueys/repogrep/internal/source"
)

func init() {
	source.Register("github", New)
}

const (
	perPage            = 100
	defaultConcurrency = 5
)

// Source implements source.Source for GitHub-starred repos.
type Source struct{}

// New constructs a GitHub source. It performs no I/O itself; auth is
// resolved lazily on the first Fetch call.
func New() (source.Source, error) {
	return &Source{}, nil
}

func (s *Source) Name() string { return "github" }

// Fetch lists the repos starred by opts.Username (or the authenticated
// user if empty), then fetches each one's README concurrently, streaming
// results as they complete.
func (s *Source) Fetch(ctx context.Context, opts source.FetchOptions) (<-chan source.FetchResult, error) {
	client, err := newClient(ctx)
	if err != nil {
		return nil, err
	}

	records, err := listStarred(ctx, client, opts)
	if err != nil {
		return nil, err
	}

	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}

	out := make(chan source.FetchResult)
	go fetchReadmes(ctx, client, records, concurrency, out)
	return out, nil
}

func listStarred(ctx context.Context, client *github.Client, opts source.FetchOptions) ([]source.RepoRecord, error) {
	listOpts := &github.ActivityListStarredOptions{ListOptions: github.ListOptions{PerPage: perPage}}

	var records []source.RepoRecord
	for {
		var starred []*github.StarredRepository
		resp, err := withRetry(ctx, func() (*github.Response, error) {
			var r *github.Response
			var e error
			starred, r, e = client.Activity.ListStarred(ctx, opts.Username, listOpts)
			return r, e
		})
		if err != nil {
			return nil, fmt.Errorf("list starred repos: %w", err)
		}

		for _, sr := range starred {
			records = append(records, mapRepo(sr.GetRepository()))
			if opts.Limit > 0 && len(records) >= opts.Limit {
				return records, nil
			}
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	return records, nil
}

// fetchReadmes fans work for records out across a worker pool, sending one
// FetchResult per record (in completion order, not input order) and
// closing out when done. A per-record README failure is reported via
// FetchResult.Err rather than aborting the run.
func fetchReadmes(ctx context.Context, client *github.Client, records []source.RepoRecord, concurrency int, out chan<- source.FetchResult) {
	defer close(out)

	jobs := make(chan int)
	go func() {
		defer close(jobs)
		for i := range records {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				rec := records[i]
				readme, path, err := fetchReadme(ctx, client, rec.Owner, rec.Name)
				result := source.FetchResult{Record: rec}
				if err != nil {
					result.Err = fmt.Errorf("fetch readme for %s: %w", rec.FullName, err)
				} else {
					result.Record.README = readme
					result.Record.ReadmePath = path
				}
				select {
				case out <- result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	wg.Wait()
}

func fetchReadme(ctx context.Context, client *github.Client, owner, name string) (content, path string, err error) {
	var rc *github.RepositoryContent
	_, err = withRetry(ctx, func() (*github.Response, error) {
		var r *github.Response
		var e error
		rc, r, e = client.Repositories.GetReadme(ctx, owner, name, nil)
		return r, e
	})
	if err != nil {
		var apiErr *github.ErrorResponse
		if errors.As(err, &apiErr) && apiErr.Response != nil && apiErr.Response.StatusCode == http.StatusNotFound {
			return "", "", nil // no README is not an error
		}
		return "", "", err
	}

	if rc.GetEncoding() == "none" {
		// GitHub returns encoding "none" (no content body) for files over
		// 1MB via this endpoint; that's not fetchable here without a
		// separate download call, and isn't worth treating as a hard
		// error — just store no README, same as a missing one.
		return "", rc.GetPath(), nil
	}

	content, err = rc.GetContent()
	if err != nil {
		return "", "", fmt.Errorf("decode readme content: %w", err)
	}
	return content, rc.GetPath(), nil
}
