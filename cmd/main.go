package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/heartwilltell/ghcontrib/internal/github"
	"github.com/heartwilltell/ghcontrib/internal/monitor"
	"github.com/heartwilltell/ghcontrib/internal/slack"
	"github.com/heartwilltell/scotty"
)

func main() {
	rootCmd := scotty.Command{
		Name:  "ghcontrib",
		Short: "GitHub organization contribution monitor",
		Long:  "Monitor GitHub users' contributions to a specified organization and send notifications to Slack",
	}

	var (
		org          string
		users        string
		githubToken  string
		slackWebhook string
		slackToken   string
		slackChannel string
		watch        bool
		intervalStr  string
		showDetails  bool
		dryRun       bool
		noCache      bool
		threshold    int
	)

	monitorCmd := scotty.Command{
		Name:  "monitor",
		Short: "Monitor GitHub contributions and send Slack notifications",
		SetFlags: func(flags *scotty.FlagSet) {
			flags.StringVarE(&org, "org", "GHCONTRIB_ORG", "",
				"GitHub organization to monitor (required)",
			)
			flags.StringVarE(&users, "users", "GHCONTRIB_USERS", "", "Comma-separated list of GitHub usernames to monitor (required)")
			flags.StringVarE(&githubToken, "github-token", "GITHUB_TOKEN", "", "GitHub personal access token (required)")
			flags.StringVarE(&slackWebhook, "slack-webhook", "SLACK_WEBHOOK_URL", "", "Slack incoming webhook URL")
			flags.StringVarE(&slackToken, "slack-token", "SLACK_BOT_TOKEN", "", "Slack bot OAuth token")
			flags.StringVarE(&slackChannel, "slack-channel", "SLACK_CHANNEL", "", "Slack channel (required with --slack-token)")
			flags.BoolVarE(&watch, "watch", "GHCONTRIB_WATCH", false, "Enable daemon/polling mode")
			flags.StringVarE(&intervalStr, "interval", "GHCONTRIB_INTERVAL", "5m", "Polling interval in watch mode (e.g., 5m, 1h)")
			flags.BoolVarE(&showDetails, "details", "GHCONTRIB_DETAILS", false, "Show detailed contribution information")
			flags.BoolVarE(&dryRun, "dry-run", "GHCONTRIB_DRY_RUN", false, "Print notifications to console instead of sending to Slack")
			flags.BoolVarE(&noCache, "no-cache", "GHCONTRIB_NO_CACHE", false, "Disable ETag caching, always fetch fresh data")
			flags.IntVarE(&threshold, "threshold", "GHCONTRIB_THRESHOLD", 0, "Minimum expected contributions per user; warns if below")
		},
		Run: func(cmd *scotty.Command, args []string) error {
			if err := validateFlags(org, users, githubToken, slackWebhook, slackToken, slackChannel, dryRun); err != nil {
				return err
			}

			userList := parseUsers(users)
			interval, err := time.ParseDuration(intervalStr)
			if err != nil {
				return fmt.Errorf("invalid interval format: %w", err)
			}

			useCache := !noCache
			githubClient := github.NewClient(githubToken, useCache)

			var slackNotifier slack.Notifier
			if dryRun {
				slackNotifier = slack.NewConsoleNotifier()
			} else if slackWebhook != "" {
				slackNotifier = slack.NewWebhookNotifier(slackWebhook)
			} else {
				slackNotifier = slack.NewBotNotifier(slackToken, slackChannel)
			}

			skipDedupe := noCache
			mon := monitor.New(githubClient, slackNotifier, org, userList, showDetails, skipDedupe, threshold)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			errChan := make(chan error, 1)

			go func() {
				if watch {
					errChan <- mon.RunWatch(ctx, interval)
				} else {
					errChan <- mon.RunOnce(ctx)
				}
			}()

			select {
			case <-sigChan:
				fmt.Println("\nReceived interrupt signal, shutting down...")
				cancel()
				return nil
			case err := <-errChan:
				return err
			}
		},
	}

	rootCmd.AddSubcommands(&monitorCmd)

	if err := rootCmd.Exec(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func validateFlags(org, users, githubToken, slackWebhook, slackToken, slackChannel string, dryRun bool) error {
	if org == "" {
		return fmt.Errorf("--org is required")
	}
	if users == "" {
		return fmt.Errorf("--users is required")
	}
	if githubToken == "" {
		return fmt.Errorf("--github-token is required")
	}

	if dryRun {
		return nil
	}

	hasWebhook := slackWebhook != ""
	hasBot := slackToken != "" && slackChannel != ""

	if !hasWebhook && !hasBot {
		return fmt.Errorf("either --slack-webhook or (--slack-token and --slack-channel) must be provided")
	}

	if slackToken != "" && slackChannel == "" {
		return fmt.Errorf("--slack-channel is required when using --slack-token")
	}

	return nil
}

func parseUsers(users string) []string {
	parts := strings.Split(users, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
