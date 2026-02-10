package github

import "time"

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Actor     Actor     `json:"actor"`
	Repo      Repo      `json:"repo"`
	Payload   Payload   `json:"payload"`
	Public    bool      `json:"public"`
	CreatedAt time.Time `json:"created_at"`
}

type Actor struct {
	ID           int    `json:"id"`
	Login        string `json:"login"`
	DisplayLogin string `json:"display_login"`
	GravatarID   string `json:"gravatar_id"`
	URL          string `json:"url"`
	AvatarURL    string `json:"avatar_url"`
}

type Repo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Payload struct {
	Action       string        `json:"action,omitempty"`
	PullRequest  *PullRequest  `json:"pull_request,omitempty"`
	Issue        *Issue        `json:"issue,omitempty"`
	Review       *Review       `json:"review,omitempty"`
	Commits      []Commit      `json:"commits,omitempty"`
	Ref          string        `json:"ref,omitempty"`
	RefType      string        `json:"ref_type,omitempty"`
	PusherType   string        `json:"pusher_type,omitempty"`
	Size         int           `json:"size,omitempty"`
	DistinctSize int           `json:"distinct_size,omitempty"`
}

type PullRequest struct {
	ID        int       `json:"id"`
	Number    int       `json:"number"`
	State     string    `json:"state"`
	Locked    bool      `json:"locked"`
	Title     string    `json:"title"`
	User      User      `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	Merged    bool      `json:"merged"`
	Mergeable *bool     `json:"mergeable,omitempty"`
	Draft     bool      `json:"draft"`
	HTMLURL   string    `json:"html_url"`
}

type Issue struct {
	ID        int       `json:"id"`
	Number    int       `json:"number"`
	State     string    `json:"state"`
	Title     string    `json:"title"`
	User      User      `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	HTMLURL   string    `json:"html_url"`
}

type Review struct {
	ID          int       `json:"id"`
	User        User      `json:"user"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	HTMLURL     string    `json:"html_url"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type Commit struct {
	SHA      string `json:"sha"`
	Author   Author `json:"author"`
	Message  string `json:"message"`
	Distinct bool   `json:"distinct"`
	URL      string `json:"url"`
}

type Author struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type User struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

const (
	EventTypePush              = "PushEvent"
	EventTypePullRequest       = "PullRequestEvent"
	EventTypePullRequestReview = "PullRequestReviewEvent"
	EventTypeIssues            = "IssuesEvent"
)
