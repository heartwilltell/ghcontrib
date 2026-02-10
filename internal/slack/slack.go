package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Notifier interface {
	Send(ctx context.Context, message Message) error
}

type WebhookNotifier struct {
	webhookURL string
	httpClient *http.Client
}

func NewWebhookNotifier(webhookURL string) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{},
	}
}

func (w *WebhookNotifier) Send(ctx context.Context, message Message) error {
	payload := map[string]any{
		"message": message.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

type BotNotifier struct {
	token      string
	channel    string
	httpClient *http.Client
}

func NewBotNotifier(token, channel string) *BotNotifier {
	return &BotNotifier{
		token:      token,
		channel:    channel,
		httpClient: &http.Client{},
	}
}

func (b *BotNotifier) Send(ctx context.Context, message Message) error {
	payload := map[string]any{
		"channel": b.channel,
		"text":    message.Text,
		"blocks":  message.Blocks,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.token))

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

type ConsoleNotifier struct{}

func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

func (c *ConsoleNotifier) Send(ctx context.Context, message Message) error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DRY RUN - Notification Preview")
	fmt.Println(strings.Repeat("=", 60))

	for _, block := range message.Blocks {
		if block.Text != nil {
			text := block.Text.Text
			text = strings.ReplaceAll(text, "*", "")
			fmt.Println(text)
		}
	}

	fmt.Println(strings.Repeat("=", 60) + "\n")
	return nil
}
