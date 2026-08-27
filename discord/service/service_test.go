package service_test

import (
	"github.com/icchon/atcoder/discord/service"
	"strings"
	"testing"
	"time"
)

type MockAtCoderClient struct {
	Submissions map[string][]service.Submission
}

func (m *MockAtCoderClient) FetchSubmissions(user string, fromSecond int64) ([]service.Submission, error) {
	return m.Submissions[user], nil
}

type MockRepository struct {
	NotifiedStatus map[string]bool
	SolvedProblems map[string]bool
	UniqueACTimes  map[string][]int64
	CurrentStreak  map[string]int
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		NotifiedStatus: make(map[string]bool),
		SolvedProblems: make(map[string]bool),
		UniqueACTimes:  make(map[string][]int64),
		CurrentStreak:  make(map[string]int),
	}
}

func (m *MockRepository) IsSuccessNotified(user, dateStr string) (bool, error) {
	return m.NotifiedStatus[user+"_"+dateStr], nil
}

func (m *MockRepository) SetSuccessNotified(user, dateStr string) error {
	m.NotifiedStatus[user+"_"+dateStr] = true
	return nil
}

func (m *MockRepository) IsProblemSolved(user, problemID string) (bool, error) {
	return m.SolvedProblems[user+"_"+problemID], nil
}

func (m *MockRepository) SaveUniqueAC(user, problemID string, epochSecond int64) error {
	m.SolvedProblems[user+"_"+problemID] = true
	m.UniqueACTimes[user] = append(m.UniqueACTimes[user], epochSecond)
	return nil
}

func (m *MockRepository) HasUniqueACSince(user string, sinceEpoch int64) (bool, error) {
	for _, t := range m.UniqueACTimes[user] {
		if t >= sinceEpoch {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockRepository) GetCurrentStreak(user string, now time.Time, jst *time.Location) (int, error) {
	return m.CurrentStreak[user], nil
}

type MockNotifier struct {
	SentMessages []string
}

func (m *MockNotifier) SendMessage(content string) error {
	m.SentMessages = append(m.SentMessages, content)
	return nil
}

type MockTimeProvider struct {
	CurrentTime time.Time
}

func (m *MockTimeProvider) Now() time.Time {
	return m.CurrentTime
}

func TestPollUniqueAC_NewACDetected(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	fixedNow := time.Date(2026, 8, 27, 14, 0, 0, 0, jst)

	cfg := service.Config{
		Users: []string{"kenkoooo"},
	}

	client := &MockAtCoderClient{
		Submissions: map[string][]service.Submission{
			"kenkoooo": {
				{ID: 1, EpochSecond: fixedNow.Unix() - 100, ProblemID: "abc300_a", Result: "AC"},
			},
		},
	}

	repo := NewMockRepository()
	repo.CurrentStreak["kenkoooo"] = 42

	notifier := &MockNotifier{}
	timeProv := &MockTimeProvider{CurrentTime: fixedNow}

	svc := service.NewWatchdogService(cfg, client, repo, notifier, timeProv, jst)

	svc.PollUniqueAC()

	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(notifier.SentMessages))
	}
	if !strings.Contains(notifier.SentMessages[0], "abc300_a") {
		t.Errorf("expected message to contain problem id, got: %s", notifier.SentMessages[0])
	}

	// 2回目のポーリング: 重複通知されない
	svc.PollUniqueAC()
	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected still 1 message, got %d", len(notifier.SentMessages))
	}
}

func TestHandleScheduledNotification_ReminderAndFinalWarning(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	fixedNow := time.Date(2026, 8, 27, 22, 0, 0, 0, jst)

	cfg := service.Config{
		Users: []string{"chokudai"},
	}

	client := &MockAtCoderClient{
		Submissions: map[string][]service.Submission{"chokudai": {}},
	}

	repo := NewMockRepository()
	repo.CurrentStreak["chokudai"] = 100

	notifier := &MockNotifier{}
	timeProv := &MockTimeProvider{CurrentTime: fixedNow}

	svc := service.NewWatchdogService(cfg, client, repo, notifier, timeProv, jst)

	svc.HandleScheduledNotification(service.NotificationSetting{Time: "22:00", Type: "reminder"})

	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(notifier.SentMessages))
	}
	if !strings.Contains(notifier.SentMessages[0], "リマインド") {
		t.Errorf("expected reminder format, got: %s", notifier.SentMessages[0])
	}

	_ = repo.SetSuccessNotified("chokudai", "2026-08-27")
	svc.HandleScheduledNotification(service.NotificationSetting{Time: "23:30", Type: "final_warning"})

	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected final warning to be skipped, got: %d", len(notifier.SentMessages))
	}
}