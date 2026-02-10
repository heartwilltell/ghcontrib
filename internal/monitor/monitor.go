package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/heartwilltell/ghcontrib/internal/github"
	"github.com/heartwilltell/ghcontrib/internal/slack"
)

type Monitor struct {
	githubClient  *github.Client
	slackNotifier slack.Notifier
	org           string
	users         []string
	showDetails   bool
	skipDedupe    bool
	seenEvents    map[string]map[string]bool
	mu            sync.Mutex
}

func New(githubClient *github.Client, slackNotifier slack.Notifier, org string, users []string, showDetails bool, skipDedupe bool) *Monitor {
	return &Monitor{
		githubClient:  githubClient,
		slackNotifier: slackNotifier,
		org:           org,
		users:         users,
		showDetails:   showDetails,
		skipDedupe:    skipDedupe,
		seenEvents:    make(map[string]map[string]bool),
	}
}

func (m *Monitor) RunOnce(ctx context.Context) error {
	log.Println("Fetching GitHub events...")

	authUser, err := m.githubClient.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}
	log.Printf("Authenticated as %s", authUser)

	events, err := m.githubClient.FetchUserOrgEvents(ctx, authUser, m.org)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	if events == nil {
		log.Println("No new events (cached)")
		return nil
	}

	if len(events) == 0 {
		log.Printf("No events found in org %s", m.org)
		return nil
	}

	filteredEvents := github.FilterRelevantEvents(events)
	log.Printf("After type filtering: %d events", len(filteredEvents))

	todayEvents := github.FilterEventsByDate(filteredEvents, time.Now())
	log.Printf("After date filtering (today): %d events", len(todayEvents))

	userMap := make(map[string]bool)
	for _, user := range m.users {
		userMap[user] = true
	}

	actorFilteredEvents := m.filterBySpecifiedUsers(todayEvents, userMap)
	log.Printf("After actor filtering: %d events", len(actorFilteredEvents))

	var allNewEvents []github.Event
	if m.skipDedupe {
		allNewEvents = actorFilteredEvents
	} else {
		for _, event := range actorFilteredEvents {
			if !m.isEventSeen(event.Actor.Login, event.ID) {
				m.markEventSeen(event.Actor.Login, event.ID)
				allNewEvents = append(allNewEvents, event)
			}
		}
	}

	if len(allNewEvents) == 0 {
		log.Println("No new events to notify")
		return nil
	}

	log.Printf("Sending notification for %d events...", len(allNewEvents))
	message := slack.FormatEventsMessage(allNewEvents, m.org, m.showDetails)
	if err := m.slackNotifier.Send(ctx, message); err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}

	log.Println("Notification sent successfully")
	return nil
}

func (m *Monitor) filterBySpecifiedUsers(events []github.Event, userMap map[string]bool) []github.Event {
	var filtered []github.Event
	for _, event := range events {
		if userMap[event.Actor.Login] {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (m *Monitor) RunWatch(ctx context.Context, interval time.Duration) error {
	log.Printf("Starting watch mode with interval %v", interval)

	if err := m.RunOnce(ctx); err != nil {
		log.Printf("Initial run failed: %v", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down monitor...")
			return ctx.Err()
		case <-ticker.C:
			if err := m.RunOnce(ctx); err != nil {
				log.Printf("Error during monitoring cycle: %v", err)
			}
		}
	}
}

func (m *Monitor) isEventSeen(user, eventID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.seenEvents[user] == nil {
		return false
	}
	return m.seenEvents[user][eventID]
}

func (m *Monitor) markEventSeen(user, eventID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.seenEvents[user] == nil {
		m.seenEvents[user] = make(map[string]bool)
	}
	m.seenEvents[user][eventID] = true

	const maxSeenEvents = 1000
	if len(m.seenEvents[user]) > maxSeenEvents {
		oldestIDs := make([]string, 0, len(m.seenEvents[user])-maxSeenEvents/2)
		for id := range m.seenEvents[user] {
			oldestIDs = append(oldestIDs, id)
			if len(oldestIDs) >= len(m.seenEvents[user])-maxSeenEvents/2 {
				break
			}
		}
		for _, id := range oldestIDs {
			delete(m.seenEvents[user], id)
		}
	}
}
