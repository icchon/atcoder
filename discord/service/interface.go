package service

import "time"

type Submission struct {
	ID          int64  `json:"id"`
	EpochSecond int64  `json:"epoch_second"`
	ProblemID   string `json:"problem_id"`
	Result      string `json:"result"`
}

type NotificationSetting struct {
	Time string `json:"time"`
	Type string `json:"type"`
}

type Config struct {
	WebhookURL          string                `json:"webhook_url"`
	Users               []string              `json:"users"`
	PollIntervalMinutes int                   `json:"poll_interval_minutes"`
	Notifications       []NotificationSetting `json:"notifications"`
}

type AtCoderClient interface {
	FetchSubmissions(user string, fromSecond int64) ([]Submission, error)
	FetchStreakCount(user string) (int, error)
}

type Repository interface {
	IsSuccessNotified(user, dateStr string) (bool, error)
	SetSuccessNotified(user, dateStr string) error
	IsProblemSolved(user, problemID string) (bool, error)
	SaveUniqueAC(user, problemID string, epochSecond int64) error
	HasUniqueACSince(user string, sinceEpoch int64) (bool, error)
}

type Notifier interface {
	SendMessage(content string) error
}

type TimeProvider interface {
	Now() time.Time
}
