package main

import (
	"github.com/icchon/atcoder/discord/service"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// --- Discord 本番送信用の Notifier ---

type RealDiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

func (d *RealDiscordNotifier) SendMessage(content string) error {
	payload := map[string]string{"content": content}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := d.client.Post(d.webhookURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook failed: %d (%s)", resp.StatusCode, string(body))
	}
	return nil
}

// --- テスト用の Mock 実装 ---

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
	NotifiedStatus map[string]bool
	SolvedProblems map[string]bool
	UniqueACTimes  map[string][]int64
	CurrentStreak  map[string]int // 追加
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		NotifiedStatus: make(map[string]bool),
		SolvedProblems: make(map[string]bool),
		UniqueACTimes:  make(map[string][]int64),
		CurrentStreak:  make(map[string]int), // 追加
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

// 追加: Repository インターフェースを満たすためのメソッド
func (m *MockRepository) GetCurrentStreak(user string, now time.Time, jst *time.Location) (int, error) {
	if val, ok := m.CurrentStreak[user]; ok {
		return val, nil
	}
	return 0, nil
}

type MockTimeProvider struct {
	CurrentTime time.Time
}

func (m *MockTimeProvider) Now() time.Time {
	return m.CurrentTime
}

// --- 実行処理 ---

func main() {
	// config.json から Webhook URL とユーザー名を取得
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config.json: %v", err)
	}
	var cfg service.Config
	if err := json.Unmarshal(configFile, &cfg); err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	if cfg.WebhookURL == "" || len(cfg.Users) == 0 {
		log.Fatal("config.json に webhook_url と users を設定してください。")
	}

	jst := time.FixedZone("JST", 9*60*60)
	targetUser := cfg.Users[0]
	notifier := &RealDiscordNotifier{
		webhookURL: cfg.WebhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}

	log.Printf("Starting mock notification test for user: %s ...\n", targetUser)

	// ==========================================
	// 1. 新規 Unique AC 達成通知のテスト
	// ==========================================
	log.Println("[1/3] Testing: Unique AC Success Notification...")
	now1 := time.Date(2026, 8, 27, 15, 30, 0, 0, jst)
	client1 := &MockAtCoderClient{
		Submissions: map[string][]service.Submission{
			targetUser: {
				{ID: 1001, EpochSecond: now1.Unix() - 60, ProblemID: "abc999_f", Result: "AC"},
			},
		},
		Streak: map[string]int{targetUser: 50},
	}
	repo1 := NewMockRepository()
	timeProv1 := &MockTimeProvider{CurrentTime: now1}

	svc1 := service.NewWatchdogService(cfg, client1, repo1, notifier, timeProv1, jst)
	svc1.PollUniqueAC()
	time.Sleep(2 * time.Second) // Discord API 制限に配慮

	// ==========================================
	// 2. 22:00 リマインド通知（未達成状態）のテスト
	// ==========================================
	log.Println("[2/3] Testing: 22:00 Reminder Notification...")
	now2 := time.Date(2026, 8, 27, 22, 0, 0, 0, jst)
	client2 := &MockAtCoderClient{
		Submissions: map[string][]service.Submission{targetUser: {}},
		Streak:      map[string]int{targetUser: 49},
	}
	repo2 := NewMockRepository()
	timeProv2 := &MockTimeProvider{CurrentTime: now2}

	svc2 := service.NewWatchdogService(cfg, client2, repo2, notifier, timeProv2, jst)
	svc2.HandleScheduledNotification(service.NotificationSetting{Time: "22:00", Type: "reminder"})
	time.Sleep(2 * time.Second)

	// ==========================================
	// 3. 23:30 最終警告通知（未達成状態）のテスト
	// ==========================================
	log.Println("[3/3] Testing: 23:30 Final Warning Notification...")
	now3 := time.Date(2026, 8, 27, 23, 30, 0, 0, jst)
	client3 := &MockAtCoderClient{
		Submissions: map[string][]service.Submission{targetUser: {}},
		Streak:      map[string]int{targetUser: 49},
	}
	repo3 := NewMockRepository()
	timeProv3 := &MockTimeProvider{CurrentTime: now3}

	svc3 := service.NewWatchdogService(cfg, client3, repo3, notifier, timeProv3, jst)
	svc3.HandleScheduledNotification(service.NotificationSetting{Time: "23:30", Type: "final_warning"})

	log.Println("All test notifications sent to Discord successfully.")
}