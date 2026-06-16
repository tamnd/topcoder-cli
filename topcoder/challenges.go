package topcoder

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ChallengesOptions controls the challenges list query.
type ChallengesOptions struct {
	// Status filters by challenge status. Accepted values: Active, Completed,
	// Draft. Empty means no filter (returns all statuses).
	Status string
	// Type filters by challenge type. Example: COMPETITIVE_PROGRAMMING.
	Type string
	// Page is the 1-indexed page number (default 1).
	Page int
	// PerPage is the number of results per page (default 20, max 50).
	PerPage int
}

// ListChallenges returns one page of TopCoder challenges matching opts.
// The second return value is the total count across all pages.
func (c *Client) ListChallenges(ctx context.Context, opts ChallengesOptions) ([]Challenge, int, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.PerPage > 50 {
		opts.PerPage = 50
	}

	u := fmt.Sprintf("%s/challenges?page=%d&perPage=%d", APIURL, opts.Page, opts.PerPage)
	if opts.Status != "" {
		u += "&status=" + strings.Title(strings.ToLower(opts.Status))
	}
	if opts.Type != "" {
		u += "&type=" + opts.Type
	}

	body, totalStr, err := c.getWithHeader(ctx, u, "X-Total")
	if err != nil {
		return nil, 0, err
	}

	var raws []rawChallenge
	if err := json.Unmarshal(body, &raws); err != nil {
		return nil, 0, fmt.Errorf("decode challenges: %w", err)
	}

	total, _ := strconv.Atoi(totalStr)
	out := make([]Challenge, len(raws))
	for i, r := range raws {
		out[i] = fromRawChallenge(r)
	}
	return out, total, nil
}

// GetChallenge fetches a single challenge by its UUID.
func (c *Client) GetChallenge(ctx context.Context, id string) (*Challenge, error) {
	u := fmt.Sprintf("%s/challenges/%s", APIURL, id)
	var raw rawChallenge
	if err := c.getJSON(ctx, u, &raw); err != nil {
		return nil, err
	}
	ch := fromRawChallenge(raw)
	return &ch, nil
}
