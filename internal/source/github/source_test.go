package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-github/v75/github"
	"github.com/hueys/repogrep/internal/source"
)

// newTestClient points a go-github client at srv instead of the real API.
func newTestClient(t *testing.T, srv *httptest.Server) *github.Client {
	t.Helper()
	client := github.NewClient(srv.Client())
	base, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	client.BaseURL = base
	return client
}

func starredJSON(t *testing.T, owner, name string) map[string]any {
	t.Helper()
	return map[string]any{
		"repo": map[string]any{
			"id":               123,
			"name":             name,
			"full_name":        owner + "/" + name,
			"owner":            map[string]any{"login": owner},
			"html_url":         "https://github.com/" + owner + "/" + name,
			"description":      "a test repo",
			"language":         "Go",
			"topics":           []string{"testing"},
			"stargazers_count": 7,
		},
	}
}

func TestListStarredPaginates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/starred", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s/user/starred?page=2>; rel="next"`, "http://"+r.Host))
			_ = json.NewEncoder(w).Encode([]any{starredJSON(t, "foo", "one")})
		case "2":
			_ = json.NewEncoder(w).Encode([]any{starredJSON(t, "foo", "two")})
		default:
			t.Fatalf("unexpected page %q", page)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv)
	records, err := listStarred(context.Background(), client, source.FetchOptions{})
	if err != nil {
		t.Fatalf("listStarred: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records across pages, got %d", len(records))
	}
	if records[0].FullName != "foo/one" || records[1].FullName != "foo/two" {
		t.Fatalf("unexpected records: %+v", records)
	}
	if records[0].Stars != 7 || records[0].PrimaryLanguage != "Go" {
		t.Fatalf("mapping lost fields: %+v", records[0])
	}
}

func TestListStarredRespectsLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/starred", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", fmt.Sprintf(`<%s/user/starred?page=2>; rel="next"`, "http://"+r.Host))
		_ = json.NewEncoder(w).Encode([]any{starredJSON(t, "foo", "one"), starredJSON(t, "foo", "two")})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv)
	records, err := listStarred(context.Background(), client, source.FetchOptions{Limit: 1})
	if err != nil {
		t.Fatalf("listStarred: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected Limit to cap results at 1, got %d", len(records))
	}
}

func TestFetchReadmeDecodesBase64AndHandles404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/foo/has-readme/readme", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content":  base64.StdEncoding.EncodeToString([]byte("# Hello\n")),
			"encoding": "base64",
			"path":     "README.md",
		})
	})
	mux.HandleFunc("/repos/foo/no-readme/readme", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
	})
	mux.HandleFunc("/repos/foo/huge-readme/readme", func(w http.ResponseWriter, r *http.Request) {
		// GitHub returns encoding "none" (no content body) for files over
		// 1MB via this endpoint.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"encoding": "none", "path": "README.md"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv)

	content, path, err := fetchReadme(context.Background(), client, "foo", "has-readme")
	if err != nil {
		t.Fatalf("fetchReadme: %v", err)
	}
	if content != "# Hello\n" || path != "README.md" {
		t.Fatalf("unexpected readme result: content=%q path=%q", content, path)
	}

	content, path, err = fetchReadme(context.Background(), client, "foo", "no-readme")
	if err != nil {
		t.Fatalf("fetchReadme on 404 should not error, got: %v", err)
	}
	if content != "" || path != "" {
		t.Fatalf("expected empty result for missing readme, got content=%q path=%q", content, path)
	}

	content, _, err = fetchReadme(context.Background(), client, "foo", "huge-readme")
	if err != nil {
		t.Fatalf("fetchReadme on >1MB (encoding=none) should not error, got: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty content for >1MB readme, got %q", content)
	}
}

func TestFetchReadmesReportsPerRecordErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/foo/ok/readme", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": base64.StdEncoding.EncodeToString([]byte("ok readme")), "encoding": "base64", "path": "README.md",
		})
	})
	mux.HandleFunc("/repos/foo/broken/readme", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity) // non-transient, non-404: exercises the Err path
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "nope"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(t, srv)
	records := []source.RepoRecord{
		{Owner: "foo", Name: "ok", FullName: "foo/ok"},
		{Owner: "foo", Name: "broken", FullName: "foo/broken"},
	}

	out := make(chan source.FetchResult)
	go fetchReadmes(context.Background(), client, records, 2, out)

	results := map[string]source.FetchResult{}
	for r := range out {
		results[r.Record.FullName] = r
	}

	if results["foo/ok"].Err != nil {
		t.Fatalf("expected foo/ok to succeed, got err: %v", results["foo/ok"].Err)
	}
	if results["foo/ok"].Record.README != "ok readme" {
		t.Fatalf("unexpected readme for foo/ok: %q", results["foo/ok"].Record.README)
	}
	if results["foo/broken"].Err == nil {
		t.Fatalf("expected foo/broken to report an error")
	}
}

func TestBackoffForClassification(t *testing.T) {
	rateErr := &github.RateLimitError{Rate: github.Rate{Reset: github.Timestamp{Time: time.Now().Add(time.Minute)}}}
	if _, retryable := backoffFor(rateErr, 0); !retryable {
		t.Errorf("RateLimitError should be retryable")
	}

	retryAfter := 2 * time.Second
	abuseErr := &github.AbuseRateLimitError{RetryAfter: &retryAfter}
	wait, retryable := backoffFor(abuseErr, 0)
	if !retryable || wait != retryAfter {
		t.Errorf("AbuseRateLimitError: got wait=%v retryable=%v, want wait=%v retryable=true", wait, retryable, retryAfter)
	}

	serverErr := &github.ErrorResponse{Response: &http.Response{StatusCode: 500}}
	if _, retryable := backoffFor(serverErr, 0); !retryable {
		t.Errorf("5xx ErrorResponse should be retryable")
	}

	notFoundErr := &github.ErrorResponse{Response: &http.Response{StatusCode: 404}}
	if _, retryable := backoffFor(notFoundErr, 0); retryable {
		t.Errorf("404 ErrorResponse should not be retryable")
	}

	if _, retryable := backoffFor(errors.New("connection reset"), 0); !retryable {
		t.Errorf("unrecognized/network-shaped errors should be retryable")
	}
}

func TestWithRetryRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	calls := 0
	_, err := withRetry(ctx, func() (*github.Response, error) {
		calls++
		return nil, errors.New("boom") // network-shaped: treated as transient
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls == 0 {
		t.Fatalf("expected fn to be called at least once before cancellation")
	}
}
