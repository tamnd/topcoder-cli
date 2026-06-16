package topcoder

import (
	"context"
	"fmt"
)

// GetMember fetches a member's public profile by handle.
func (c *Client) GetMember(ctx context.Context, handle string) (*Member, error) {
	u := fmt.Sprintf("%s/members/%s", APIURL, handle)
	var raw rawMember
	if err := c.getJSON(ctx, u, &raw); err != nil {
		return nil, err
	}
	m := fromRawMember(raw)
	return &m, nil
}
