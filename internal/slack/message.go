package slack

import (
	"fmt"
	"strings"

	"github.com/heartwilltell/ghcontrib/internal/github"
)

type Message struct {
	Text   string  `json:"text"`
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Type      string      `json:"type"`
	Text      *TextBlock  `json:"text,omitempty"`
	Fields    []TextBlock `json:"fields,omitempty"`
	Accessory *Accessory  `json:"accessory,omitempty"`
}

type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Accessory struct {
	Type     string `json:"type"`
	ImageURL string `json:"image_url"`
	AltText  string `json:"alt_text"`
}

func FormatEventsMessage(events []github.Event, org string, showDetails bool) Message {
	if len(events) == 0 {
		return Message{}
	}

	userEvents := make(map[string][]github.Event)

	for _, event := range events {
		userEvents[event.Actor.Login] = append(userEvents[event.Actor.Login], event)
	}

	var blocks []Block
	var textParts []string

	for user, userEventList := range userEvents {
		blocks = append(blocks, formatUserSummary(user, userEventList, showDetails)...)
		textParts = append(textParts, formatUserText(user, userEventList, showDetails))
	}

	return Message{
		Text:   strings.Join(textParts, "\n\n"),
		Blocks: blocks,
	}
}

func formatUserText(user string, events []github.Event, showDetails bool) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("*%s* made %d contribution(s) today", user, len(events)))

	if showDetails {
		for _, event := range events {
			lines = append(lines, formatEventText(event))
		}
	}

	return strings.Join(lines, "\n")
}

func formatEventText(event github.Event) string {
	switch event.Type {
	case github.EventTypePush:
		commitCount := event.Payload.Size
		if commitCount == 0 {
			commitCount = len(event.Payload.Commits)
		}
		branch := strings.TrimPrefix(event.Payload.Ref, "refs/heads/")
		return fmt.Sprintf("  • Pushed %d commit(s) to `%s` in %s",
			commitCount, branch, event.Repo.Name)

	case github.EventTypePullRequest:
		action := event.Payload.Action
		if action == "closed" && event.Payload.PullRequest.Merged {
			action = "merged"
		}
		title := event.Payload.PullRequest.Title
		if title == "" {
			title = "(no title)"
		}
		return fmt.Sprintf("  • %s PR #%d: %s in %s",
			strings.Title(action),
			event.Payload.PullRequest.Number,
			title,
			event.Repo.Name)

	case github.EventTypePullRequestReview:
		return fmt.Sprintf("  • Reviewed PR (%s) in %s",
			event.Payload.Review.State,
			event.Repo.Name)

	case github.EventTypeIssues:
		return fmt.Sprintf("  • %s issue #%d: %s in %s",
			strings.Title(event.Payload.Action),
			event.Payload.Issue.Number,
			event.Payload.Issue.Title,
			event.Repo.Name)

	default:
		return fmt.Sprintf("  • %s in %s", event.Type, event.Repo.Name)
	}
}

func formatUserSummary(user string, events []github.Event, showDetails bool) []Block {
	var blocks []Block

	count := len(events)
	summary := fmt.Sprintf("User *%s* made %d contribution(s) today", user, count)

	blocks = append(blocks, Block{
		Type: "section",
		Text: &TextBlock{
			Type: "mrkdwn",
			Text: summary,
		},
	})

	if showDetails {
		for _, event := range events {
			blocks = append(blocks, formatEventDetail(event))
		}
	}

	return blocks
}

func formatEventDetail(event github.Event) Block {
	var text string

	switch event.Type {
	case github.EventTypePush:
		commitCount := event.Payload.Size
		if commitCount == 0 {
			commitCount = len(event.Payload.Commits)
		}
		branch := strings.TrimPrefix(event.Payload.Ref, "refs/heads/")
		text = fmt.Sprintf("  • Pushed %d commit(s) to `%s` in %s",
			commitCount, branch, event.Repo.Name)

	case github.EventTypePullRequest:
		action := event.Payload.Action
		if action == "closed" && event.Payload.PullRequest.Merged {
			action = "merged"
		}
		title := event.Payload.PullRequest.Title
		if title == "" {
			title = "(no title)"
		}
		text = fmt.Sprintf("  • %s PR #%d: %s in %s",
			strings.Title(action),
			event.Payload.PullRequest.Number,
			title,
			event.Repo.Name)

	case github.EventTypePullRequestReview:
		state := event.Payload.Review.State
		text = fmt.Sprintf("  • Reviewed PR (%s) in %s",
			state,
			event.Repo.Name)

	case github.EventTypeIssues:
		action := event.Payload.Action
		text = fmt.Sprintf("  • %s issue #%d: %s in %s",
			strings.Title(action),
			event.Payload.Issue.Number,
			event.Payload.Issue.Title,
			event.Repo.Name)

	default:
		text = fmt.Sprintf("  • %s in %s", event.Type, event.Repo.Name)
	}

	return Block{
		Type: "section",
		Text: &TextBlock{
			Type: "mrkdwn",
			Text: text,
		},
	}
}
