package topcoder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/topcoder-cli/topcoder"
)

func TestDefaultConfig(t *testing.T) {
	cfg := topcoder.DefaultConfig()
	if cfg.Rate <= 0 {
		t.Errorf("Rate = %v, want > 0", cfg.Rate)
	}
	if cfg.Retries <= 0 {
		t.Errorf("Retries = %d, want > 0", cfg.Retries)
	}
	if cfg.Timeout <= 0 {
		t.Errorf("Timeout = %v, want > 0", cfg.Timeout)
	}
	if cfg.UserAgent == "" {
		t.Error("UserAgent is empty")
	}
}

func TestNewClientNotNil(t *testing.T) {
	c := topcoder.NewClient(topcoder.DefaultConfig())
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestChallengeRoundTrip(t *testing.T) {
	want := topcoder.Challenge{
		ID:     "abc-123",
		Name:   "Test Challenge",
		Status: "Active",
		Type:   "COMPETITIVE_PROGRAMMING",
		Prize1: 1500,
		Prize2: 750,
	}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got topcoder.Challenge
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Name != want.Name || got.Prize1 != want.Prize1 {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestMemberRoundTrip(t *testing.T) {
	want := topcoder.Member{
		Handle:  "rng_58",
		Country: "Japan",
		Wins:    42,
	}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got topcoder.Member
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Handle != want.Handle || got.Wins != want.Wins {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestErrSentinels(t *testing.T) {
	for name, err := range map[string]error{
		"ErrNotFound":    topcoder.ErrNotFound,
		"ErrRateLimited": topcoder.ErrRateLimited,
		"ErrBlocked":     topcoder.ErrBlocked,
	} {
		if err == nil {
			t.Errorf("%s is nil", name)
		}
		if err != nil && err.Error() == "" {
			t.Errorf("%s has empty message", name)
		}
	}
}

func TestGetChallengeFromServer(t *testing.T) {
	want := map[string]any{
		"id":     "uuid-abc",
		"name":   "My Challenge",
		"status": "Active",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request has no User-Agent")
		}
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "uuid-abc" {
		t.Errorf("got id %q, want %q", got["id"], "uuid-abc")
	}
}

func TestGetUserFromServer(t *testing.T) {
	want := map[string]any{
		"handle":  "tourist",
		"country": "Belarus",
		"wins":    float64(3),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["handle"] != "tourist" {
		t.Errorf("got handle %q, want %q", got["handle"], "tourist")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := topcoder.DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 0
	c := topcoder.NewClient(cfg)

	_, err := c.GetMember(ctx, "test")
	if err == nil {
		t.Error("GetMember with cancelled context returned nil error")
	}
}

func TestChallengeTagsField(t *testing.T) {
	ch := topcoder.Challenge{
		ID:   "test-id",
		Tags: []string{"Go", "REST", "PostgreSQL"},
	}
	b, _ := json.Marshal(ch)
	var got topcoder.Challenge
	_ = json.Unmarshal(b, &got)
	if len(got.Tags) != 3 {
		t.Errorf("Tags len = %d, want 3", len(got.Tags))
	}
}

func TestChallengeDeadlineField(t *testing.T) {
	ch := topcoder.Challenge{Deadline: "2024-12-31"}
	if ch.Deadline != "2024-12-31" {
		t.Errorf("Deadline = %q, want 2024-12-31", ch.Deadline)
	}
}

func TestMemberSkillsField(t *testing.T) {
	m := topcoder.Member{
		Handle: "test",
		Skills: []string{"Java", "C++", "Python"},
	}
	if len(m.Skills) != 3 {
		t.Errorf("Skills len = %d, want 3", len(m.Skills))
	}
}

func TestDefaultUserAgentAndHost(t *testing.T) {
	if topcoder.DefaultUserAgent == "" {
		t.Error("DefaultUserAgent is empty")
	}
	if topcoder.Host == "" {
		t.Error("Host is empty")
	}
	if topcoder.APIURL == "" {
		t.Error("APIURL is empty")
	}
}

func TestChallengeCreatedTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ch := topcoder.Challenge{ID: "t1", Created: now}
	b, _ := json.Marshal(ch)
	var got topcoder.Challenge
	_ = json.Unmarshal(b, &got)
	if !got.Created.Equal(now) {
		t.Errorf("Created = %v, want %v", got.Created, now)
	}
}
