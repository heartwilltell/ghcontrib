package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"
)

type Client struct {
	httpClient *http.Client
	token      string
	useCache   bool
	etagCache  map[string]string
	mu         sync.RWMutex
}

func NewClient(token string, useCache bool) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      token,
		useCache:   useCache,
		etagCache:  make(map[string]string),
	}
}

func (c *Client) GetAuthenticatedUser(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return user.Login, nil
}

const maxPages = 10

var nextLinkRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

func parseNextLink(header string) string {
	matches := nextLinkRe.FindStringSubmatch(header)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func (c *Client) FetchUserOrgEvents(ctx context.Context, user, org string) ([]Event, error) {
	firstURL := fmt.Sprintf("https://api.github.com/users/%s/events/orgs/%s", user, org)

	if c.useCache {
		c.mu.RLock()
		etag, hasEtag := c.etagCache[firstURL]
		c.mu.RUnlock()

		if hasEtag {
			events, nextURL, notModified, err := c.fetchEventsPage(ctx, firstURL, etag)
			if err != nil {
				return nil, err
			}
			if notModified {
				return nil, nil
			}

			allEvents := events
			url := nextURL
			for page := 1; page < maxPages && url != ""; page++ {
				pageEvents, next, _, err := c.fetchEventsPage(ctx, url, "")
				if err != nil {
					return nil, err
				}
				allEvents = append(allEvents, pageEvents...)
				url = next
			}

			log.Printf("Fetched %d total events for user %s in org %s", len(allEvents), user, org)
			return allEvents, nil
		}
	}

	var allEvents []Event
	url := firstURL
	for page := 0; page < maxPages && url != ""; page++ {
		events, nextURL, _, err := c.fetchEventsPage(ctx, url, "")
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, events...)
		url = nextURL
	}

	log.Printf("Fetched %d total events for user %s in org %s", len(allEvents), user, org)
	return allEvents, nil
}

func (c *Client) fetchEventsPage(ctx context.Context, url, etag string) ([]Event, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, "", true, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", false, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	if c.useCache {
		if newEtag := resp.Header.Get("ETag"); newEtag != "" {
			c.mu.Lock()
			c.etagCache[url] = newEtag
			c.mu.Unlock()
		}
	}

	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	nextURL := parseNextLink(resp.Header.Get("Link"))
	return events, nextURL, false, nil
}

func FilterRelevantEvents(events []Event) []Event {
	filtered := make([]Event, 0)
	for _, event := range events {
		switch event.Type {
		case EventTypePush, EventTypePullRequest, EventTypePullRequestReview, EventTypeIssues:
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func FilterEventsByOrg(events []Event, org string) []Event {
	filtered := make([]Event, 0)
	orgPrefix := org + "/"
	for _, event := range events {
		if len(event.Repo.Name) > len(orgPrefix) && event.Repo.Name[:len(orgPrefix)] == orgPrefix {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func FilterEventsByDate(events []Event, targetDate time.Time) []Event {
	filtered := make([]Event, 0)
	year, month, day := targetDate.Date()

	for _, event := range events {
		eventYear, eventMonth, eventDay := event.CreatedAt.Date()
		if year == eventYear && month == eventMonth && day == eventDay {
			filtered = append(filtered, event)
		}
	}
	return filtered
}
