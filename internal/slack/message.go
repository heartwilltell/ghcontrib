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

func displayName(ghUser string, slackUsers map[string]string) string {
	if slackID, ok := slackUsers[ghUser]; ok {
		return fmt.Sprintf("<@%s>", slackID)
	}
	return ghUser
}

func FormatEventsMessage(events []github.Event, org string, showDetails bool, threshold int, users []string, slackUsers map[string]string) Message {
	userEvents := make(map[string][]github.Event)

	for _, event := range events {
		userEvents[event.Actor.Login] = append(userEvents[event.Actor.Login], event)
	}

	var blocks []Block
	var textParts []string

	for user, userEventList := range userEvents {
		name := displayName(user, slackUsers)
		blocks = append(blocks, formatUserSummary(name, userEventList, showDetails)...)
		textParts = append(textParts, formatUserText(name, userEventList, showDetails))
	}

	if threshold > 0 {
		for _, user := range users {
			count := len(userEvents[user])
			if count >= threshold {
				continue
			}
			name := displayName(user, slackUsers)
			warning := fmt.Sprintf(":warning: %s made fewer than %d contributions today. Please improve your stats.", name, threshold)
			textParts = append(textParts, warning)
			blocks = append(blocks, Block{
				Type: "section",
				Text: &TextBlock{
					Type: "mrkdwn",
					Text: warning,
				},
			})
		}
	}

	return Message{
		Text:   strings.Join(textParts, "\n\n"),
		Blocks: blocks,
	}
}

func formatUserText(name string, events []github.Event, showDetails bool) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("%s made %d contribution(s) today", name, len(events)))

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
		branch := strings.TrimPrefix(event.Payload.Ref, "refs/heads/")
		commitCount := event.Payload.Size
		if commitCount == 0 {
			commitCount = len(event.Payload.Commits)
		}
		if commitCount > 0 {
			return fmt.Sprintf("  - Pushed %d commit(s) to %s in %s",
				commitCount, branch, event.Repo.Name)
		}
		return fmt.Sprintf("  - Pushed to %s in %s", branch, event.Repo.Name)

	case github.EventTypePullRequest:
		action := event.Payload.Action
		if action == "closed" && event.Payload.PullRequest.Merged {
			action = "merged"
		}
		title := event.Payload.PullRequest.Title
		if title == "" {
			title = "(no title)"
		}
		return fmt.Sprintf("  - %s PR #%d: %s in %s",
			strings.Title(action),
			event.Payload.PullRequest.Number,
			title,
			event.Repo.Name)

	case github.EventTypePullRequestReview:
		return fmt.Sprintf("  - Reviewed PR (%s) in %s",
			event.Payload.Review.State,
			event.Repo.Name)

	case github.EventTypeIssues:
		return fmt.Sprintf("  - %s issue #%d: %s in %s",
			strings.Title(event.Payload.Action),
			event.Payload.Issue.Number,
			event.Payload.Issue.Title,
			event.Repo.Name)

	default:
		return fmt.Sprintf("  - %s in %s", event.Type, event.Repo.Name)
	}
}

func formatUserSummary(name string, events []github.Event, showDetails bool) []Block {
	var blocks []Block

	count := len(events)
	summary := fmt.Sprintf("%s made %d contribution(s) today", name, count)

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
		branch := strings.TrimPrefix(event.Payload.Ref, "refs/heads/")
		commitCount := event.Payload.Size
		if commitCount == 0 {
			commitCount = len(event.Payload.Commits)
		}
		if commitCount > 0 {
			text = fmt.Sprintf("  • Pushed %d commit(s) to `%s` in %s",
				commitCount, branch, event.Repo.Name)
		} else {
			text = fmt.Sprintf("  • Pushed to `%s` in %s", branch, event.Repo.Name)
		}

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
