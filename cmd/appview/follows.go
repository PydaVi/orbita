package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// The feed reuses the existing Bluesky follow graph instead of inventing a
// parallel "follow" concept for this product. app.bsky.graph.follow is a
// public collection on the follower's own repo — read straight from their
// PDS, same as app.bsky.actor.profile already is for avatar/bio, no
// Bluesky-specific API involved.
func fetchFollowedDIDs(ctx context.Context, pdsURL, did string) ([]string, error) {
	var dids []string
	cursor := ""
	for {
		u := fmt.Sprintf("%s/xrpc/com.atproto.repo.listRecords?repo=%s&collection=app.bsky.graph.follow&limit=100",
			pdsURL, url.QueryEscape(did))
		if cursor != "" {
			u += "&cursor=" + url.QueryEscape(cursor)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		var body struct {
			Records []struct {
				Value struct {
					Subject string `json:"subject"`
				} `json:"value"`
			} `json:"records"`
			Cursor string `json:"cursor"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("PDS returned %d listing follows", resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}

		for _, rec := range body.Records {
			if rec.Value.Subject != "" {
				dids = append(dids, rec.Value.Subject)
			}
		}

		if body.Cursor == "" || len(body.Records) == 0 {
			break
		}
		cursor = body.Cursor
	}
	return dids, nil
}
