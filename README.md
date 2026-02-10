# ghcontrib

GitHub organization contribution monitor with Slack notifications.

## Overview

`ghcontrib` monitors GitHub users' contributions to a specified organization and sends formatted notifications to Slack. It tracks **today's** commits, pull requests, code reviews, and issues.

Built with [scotty](https://github.com/heartwilltell/scotty) CLI framework.

## Features

- Monitor multiple GitHub users' activity in a specific organization
- **Works with both org members and independent contributors**
- Track commits (PushEvent), pull requests, code reviews, and issues
- Send formatted Slack notifications via webhook or bot token
- One-shot execution or continuous monitoring (watch mode)
- Efficient ETag caching to minimize API calls (optional, can be disabled)
- Zero dependencies beyond scotty (pure standard library)

## Quick Start with GitHub Actions

1. Fork or clone this repository to your GitHub account
2. Go to **Settings** → **Secrets and variables** → **Actions**
3. Add these secrets:
   - `GHCONTRIB_ORG` - Your GitHub organization
   - `GHCONTRIB_USERS` - Comma-separated usernames
   - `GH_CONTRIB_TOKEN` - GitHub token with `read:org` scope
   - `SLACK_WEBHOOK_URL` - Slack webhook URL
4. The workflow will run automatically on weekdays at 16:00 CET
5. Or trigger manually from the **Actions** tab

See [GitHub Actions Setup](#github-actions-recommended-for-daily-reports) for detailed instructions.

## Installation

### From Source

```bash
go install github.com/heartwilltell/ghcontrib/cmd/ghcontrib@latest
```

### Manual Build

```bash
git clone https://github.com/heartwilltell/ghcontrib.git
cd ghcontrib
go build -o ghcontrib ./cmd
```

## Usage

### Command Structure

```bash
ghcontrib monitor [flags]
```

### Flags

All flags support environment variable fallback:

| Flag | Environment Variable | Required | Description |
|------|---------------------|----------|-------------|
| `--org` | `GHCONTRIB_ORG` | yes | GitHub organization to monitor |
| `--users` | `GHCONTRIB_USERS` | yes | Comma-separated GitHub usernames |
| `--github-token` | `GITHUB_TOKEN` | yes | GitHub personal access token |
| `--slack-webhook` | `SLACK_WEBHOOK_URL` | no* | Slack incoming webhook URL |
| `--slack-token` | `SLACK_BOT_TOKEN` | no* | Slack bot OAuth token |
| `--slack-channel` | `SLACK_CHANNEL` | no* | Slack channel (required with `--slack-token`) |
| `--watch` | `GHCONTRIB_WATCH` | no | Enable daemon/polling mode (default: false) |
| `--interval` | `GHCONTRIB_INTERVAL` | no | Polling interval (default: `5m`) |
| `--details` | `GHCONTRIB_DETAILS` | no | Show detailed contribution info (default: false) |
| `--dry-run` | `GHCONTRIB_DRY_RUN` | no | Print to console instead of Slack (default: false) |
| `--no-cache` | `GHCONTRIB_NO_CACHE` | no | Disable ETag caching, always fetch fresh data (default: false) |

\* At least one Slack notification method must be configured (webhook or bot), unless using `--dry-run`.

### Examples

#### One-shot execution with webhook (summary only)

```bash
ghcontrib monitor \
  --org myorg \
  --users "user1,user2,user3" \
  --github-token ghp_xxxxxxxxxxxxx \
  --slack-webhook https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX
```

#### Continuous monitoring with bot token and detailed output

```bash
ghcontrib monitor \
  --org myorg \
  --users "user1,user2" \
  --github-token ghp_xxxxxxxxxxxxx \
  --slack-token xoxb-xxxxxxxxxxxxx \
  --slack-channel "#github-activity" \
  --watch \
  --interval 10m \
  --details
```

#### Using environment variables

```bash
export GHCONTRIB_ORG=myorg
export GHCONTRIB_USERS=user1,user2,user3
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...

ghcontrib monitor --watch
```

#### Dry-run mode (testing without sending to Slack)

```bash
ghcontrib monitor \
  --org myorg \
  --users "user1,user2" \
  --github-token ghp_xxxxxxxxxxxxx \
  --dry-run \
  --details
```

This will fetch GitHub events and print the notification to your console instead of sending to Slack. Perfect for testing your configuration!

#### Disable caching (always fetch fresh data)

```bash
ghcontrib monitor \
  --org myorg \
  --users "user1,user2" \
  --github-token ghp_xxxxxxxxxxxxx \
  --slack-webhook xxx \
  --no-cache
```

By default, the tool uses ETag caching to avoid refetching unchanged data (saves API rate limit). Use `--no-cache` to always fetch fresh data, useful for:
- Testing and debugging
- Ensuring you see the absolute latest events
- When you don't care about API rate limits

## Notification Format

The tool sends plain summary notifications to Slack. Use the `--details` flag to include contribution details.

### Summary Mode (default)

Shows count of contributions made **today**:

```
User john-doe made 5 contribution(s) today

User jane-smith made 3 contribution(s) today

User alice-dev made 2 contribution(s) today
```

### Detailed Mode (with `--details` flag)

```
User john-doe made 5 contribution(s) today
  • Pushed 3 commit(s) to main in myorg/backend-api
  • Opened PR #142: Add user authentication in myorg/backend-api
  • Reviewed PR (approved) in myorg/frontend

User jane-smith made 3 contribution(s) today
  • Merged PR #138: Fix database connection pool in myorg/backend-api
  • Pushed 2 commit(s) to feature/oauth in myorg/auth-service
  • Opened issue #45: Update documentation in myorg/backend-api

User alice-dev made 2 contribution(s) today
  • Reviewed PR (commented) in myorg/backend-api
  • Closed issue #42: Fix login bug in myorg/frontend
```

### Dry-Run Mode (with `--dry-run` flag)

When using `--dry-run`, notifications are printed to the console instead of sent to Slack:

```
============================================================
DRY RUN - Notification Preview
============================================================
User john-doe made 5 contribution(s) today
  • Pushed 3 commit(s) to main in myorg/backend-api
  • Opened PR #142: Add user authentication in myorg/backend-api
  • Reviewed PR (approved) in myorg/frontend
User jane-smith made 3 contribution(s) today
  • Merged PR #138: Fix database connection pool in myorg/backend-api
  • Pushed 2 commit(s) to feature/oauth in myorg/auth-service
============================================================
```

This is perfect for testing your configuration before connecting to Slack!

## How It Works

The tool monitors GitHub contributions by:

1. **Fetching public events** for each specified user via GitHub's `/users/{username}/events/public` API (up to 90 days)
2. **Filtering by date** - Only events from **today** (based on event creation time)
3. **Filtering by organization** - Only events from repositories in the specified org are included
4. **Filtering by event type** - Only tracks commits, PRs, reviews, and issues
5. **Filtering by actor** - Only shows contributions from the users you specified
6. **Deduplicating** - Tracks seen events to avoid duplicate notifications (unless `--no-cache` is used)

**Important:** This approach works for:
- ✅ Organization members
- ✅ External/independent contributors
- ✅ Contributors who aren't part of the org but contribute to public repos
- ✅ Only shows **today's contributions** (not historical)

The tool only accesses **public events**, so private repository activity won't be tracked unless you have appropriate access.

## Configuration

### GitHub Token

Create a GitHub personal access token with the following scopes:
- `public_repo` - Access public repositories (for reading public events)
- `read:user` - Read user profile data (optional, improves rate limits)

**Note:** The tool fetches public events for each user and filters by organization, so it works for both:
- Organization members
- Independent contributors (external collaborators)

Generate at: https://github.com/settings/tokens

### Slack Webhook

1. Go to https://api.slack.com/apps
2. Create a new app or select existing
3. Enable "Incoming Webhooks"
4. Add new webhook to workspace
5. Copy the webhook URL

### Slack Bot Token

1. Go to https://api.slack.com/apps
2. Create a new app or select existing
3. Add OAuth scopes: `chat:write`, `chat:write.public`
4. Install app to workspace
5. Copy the Bot User OAuth Token (starts with `xoxb-`)
6. Invite the bot to your channel: `/invite @YourBotName`

## Running in Production

### GitHub Actions (Recommended for Daily Reports)

The repository includes a GitHub Actions workflow that runs automatically on weekdays at 16:00 CET.

#### Setup

1. **Configure Repository Secrets** (Settings → Secrets and variables → Actions → New repository secret):
   - `GHCONTRIB_ORG` - Your GitHub organization name
   - `GHCONTRIB_USERS` - Comma-separated list of usernames (e.g., `user1,user2,user3`)
   - `GH_CONTRIB_TOKEN` - GitHub personal access token with `read:org` scope
   - `SLACK_WEBHOOK_URL` - Your Slack incoming webhook URL

2. **Optional: Configure Repository Variables** (Settings → Secrets and variables → Actions → Variables):
   - `GHCONTRIB_DETAILS` - Set to `true` to show detailed contributions (default: `false`)

3. **Manual Trigger**: You can also trigger the workflow manually from the Actions tab.

The workflow file is located at `.github/workflows/daily-report.yml`.

#### Schedule Notes

- The workflow runs at **15:00 UTC** which corresponds to:
  - **16:00 CET** (Central European Time - winter)
  - **17:00 CEST** (Central European Summer Time - summer)
- Runs **Monday through Friday** (weekdays only)
- To adjust the time, edit the cron expression in `.github/workflows/daily-report.yml`:
  ```yaml
  # Format: 'minute hour day month weekday'
  # For 14:00 UTC (15:00 CET winter / 16:00 CEST summer):
  - cron: '0 14 * * 1-5'
  ```

#### Testing the Workflow

Before the scheduled run, test the workflow manually:

1. Go to your repository on GitHub
2. Click **Actions** tab
3. Select **Daily GitHub Contributions Report** workflow
4. Click **Run workflow** → **Run workflow**
5. Monitor the run and check logs for any errors

#### Alternative: Using Slack Bot Token

If you prefer using a Slack bot token instead of webhook, modify the workflow:

```yaml
env:
  GHCONTRIB_ORG: ${{ secrets.GHCONTRIB_ORG }}
  GHCONTRIB_USERS: ${{ secrets.GHCONTRIB_USERS }}
  GITHUB_TOKEN: ${{ secrets.GH_CONTRIB_TOKEN }}
  SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
  SLACK_CHANNEL: ${{ secrets.SLACK_CHANNEL }}
  GHCONTRIB_DETAILS: ${{ vars.GHCONTRIB_DETAILS || 'false' }}
```

### Systemd Service

Create `/etc/systemd/system/ghcontrib.service`:

```ini
[Unit]
Description=GitHub Contribution Monitor
After=network.target

[Service]
Type=simple
User=youruser
Environment="GHCONTRIB_ORG=myorg"
Environment="GHCONTRIB_USERS=user1,user2"
Environment="GITHUB_TOKEN=ghp_xxxxxxxxxxxxx"
Environment="SLACK_WEBHOOK_URL=https://hooks.slack.com/services/..."
ExecStart=/usr/local/bin/ghcontrib monitor --watch --interval 5m
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable ghcontrib
sudo systemctl start ghcontrib
sudo systemctl status ghcontrib
```

### Cron Job (One-shot mode)

Add to crontab (`crontab -e`):

```cron
*/5 * * * * /usr/local/bin/ghcontrib monitor --org myorg --users user1,user2 --github-token "$GITHUB_TOKEN" --slack-webhook "$SLACK_WEBHOOK_URL" >> /var/log/ghcontrib.log 2>&1
```

## Development

### Project Structure

```
ghcontrib/
├── .github/
│   └── workflows/
│       └── daily-report.yml  # GitHub Actions workflow
├── cmd/                      # CLI entry point
│   └── main.go
├── internal/
│   ├── github/               # GitHub API client
│   │   ├── client.go
│   │   └── types.go
│   ├── slack/                # Slack notifier
│   │   ├── slack.go
│   │   └── message.go
│   └── monitor/              # Core monitoring logic
│       └── monitor.go
├── go.mod
└── README.md
```

### Build

```bash
go build -o ghcontrib ./cmd/ghcontrib
```

### Run Tests

```bash
go test ./...
```

## License

MIT License - see [LICENSE](LICENSE) file for details.