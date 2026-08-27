package service_test

import (
	"github.com/icchon/atcoder/discord/service"
	"strings"
	"testing"
	"time"
)

// --- Mock 定義 ---

type MockAtCoderClient struct {
	Submissions map[string][]service.Submission
	Streak      map[string]int
}

func (m *MockAtCoderClient) FetchSubmissions(user string, fromSecond int64) ([]service.Submission, error) {
	return m.Submissions[user], nil
}

func (m *MockAtCoderClient) FetchStreakCount(user string) (int, error) {
	return m.Streak[user], nil
}

type MockRepository struct {
	NotifiedStatus map[string]bool     // "user_date" -> bool
	SolvedProblems map[string]bool     // "user_problem" -> bool
	UniqueACTimes  map[string][]int64  // user -> []epoch
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		NotifiedStatus: make(map[string]bool),
		SolvedProblems: make(map[string]bool),
		UniqueACTimes:  make(map[string][]int64),
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

// --- テスト本体 ---

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
		Streak: map[string]int{"kenkoooo": 42},
	}

	repo := NewMockRepository()
	notifier := &MockNotifier{}
	timeProv := &MockTimeProvider{CurrentTime: fixedNow}

	svc := service.NewWatchdogService(cfg, client, repo, notifier, timeProv, jst)

	// 1回目のポーリング: 新規 Unique AC を検知して通知される
	svc.PollUniqueAC()

	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(notifier.SentMessages))
	}
	if !strings.Contains(notifier.SentMessages[0], "abc300_a") {
		t.Errorf("expected message to contain problem id, got: %s", notifier.SentMessages[0])
	}

	// 2回目のポーリング: 同日中に再度呼ばれても重複通知されない
	svc.PollUniqueAC()
	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected still 1 message (no duplicate), got %d", len(notifier.SentMessages))
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
		Streak:      map[string]int{"chokudai": 100},
	}

	repo := NewMockRepository()
	notifier := &MockNotifier{}
	timeProv := &MockTimeProvider{CurrentTime: fixedNow}

	svc := service.NewWatchdogService(cfg, client, repo, notifier, timeProv, jst)

	// リマインド通知のテスト (未達成時)
	svc.HandleScheduledNotification(service.NotificationSetting{Time: "22:00", Type: "reminder"})

	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(notifier.SentMessages))
	}
	if !strings.Contains(notifier.SentMessages[0], "リマインド") {
		t.Errorf("expected reminder format, got: %s", notifier.SentMessages[0])
	}

	// 達成フラグを立てた後 -> 最終通告がスキップされることを確認
	_ = repo.SetSuccessNotified("chokudai", "2026-08-27")
	svc.HandleScheduledNotification(service.NotificationSetting{Time: "23:30", Type: "final_warning"})

	if len(notifier.SentMessages) != 1 {
		t.Fatalf("expected final warning to be skipped, message count: %d", len(notifier.SentMessages))
	}
}
