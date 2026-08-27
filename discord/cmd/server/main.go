package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/icchon/atcoder/discord/service"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
)

// --- 具象実装 ---

type RealAtCoderClient struct {
	client *http.Client
}

func (c *RealAtCoderClient) FetchSubmissions(user string, fromSecond int64) ([]service.Submission, error) {
	url := fmt.Sprintf("https://kenkoooo.com/atcoder/atcoder-api/v3/user/submissions?user=%s&from_second=%d", user, fromSecond)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var subs []service.Submission
	if err := json.Unmarshal(body, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

type SQLiteRepository struct {
	db *sql.DB
}

func (r *SQLiteRepository) IsSuccessNotified(user, dateStr string) (bool, error) {
	var notified int
	err := r.db.QueryRow("SELECT notified_success FROM daily_status WHERE atcoder_id = ? AND date_jst = ?", user, dateStr).Scan(&notified)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return notified == 1, err
}

func (r *SQLiteRepository) SetSuccessNotified(user, dateStr string) error {
	query := `
	INSERT INTO daily_status (atcoder_id, date_jst, notified_success)
	VALUES (?, ?, 1)
	ON CONFLICT(atcoder_id, date_jst) DO UPDATE SET notified_success = 1;`
	_, err := r.db.Exec(query, user, dateStr)
	return err
}

func (r *SQLiteRepository) IsProblemSolved(user, problemID string) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM unique_ac WHERE atcoder_id = ? AND problem_id = ?", user, problemID).Scan(&count)
	return count > 0, err
}

func (r *SQLiteRepository) SaveUniqueAC(user, problemID string, epochSecond int64) error {
	_, err := r.db.Exec("INSERT INTO unique_ac (atcoder_id, problem_id, first_ac_epoch) VALUES (?, ?, ?)", user, problemID, epochSecond)
	return err
}

func (r *SQLiteRepository) HasUniqueACSince(user string, sinceEpoch int64) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM unique_ac WHERE atcoder_id = ? AND first_ac_epoch >= ?", user, sinceEpoch).Scan(&count)
	return count > 0, err
}

// unique_ac テーブルから JST 基準の Current Streak を算出
func (r *SQLiteRepository) GetCurrentStreak(user string, now time.Time, jst *time.Location) (int, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT date(first_ac_epoch, 'unixepoch', '+9 hours') AS jst_date
		FROM unique_ac
		WHERE atcoder_id = ?
	`, user)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	solvedDates := make(map[string]bool)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil {
			solvedDates[d] = true
		}
	}

	today := now.In(jst)
	todayStr := today.Format("2006-01-02")
	yesterdayStr := today.AddDate(0, 0, -1).Format("2006-01-02")

	var currentCursor time.Time
	if solvedDates[todayStr] {
		currentCursor = today
	} else if solvedDates[yesterdayStr] {
		currentCursor = today.AddDate(0, 0, -1)
	} else {
		return 0, nil
	}

	streak := 0
	for {
		dateStr := currentCursor.Format("2006-01-02")
		if !solvedDates[dateStr] {
			break
		}
		streak++
		currentCursor = currentCursor.AddDate(0, 0, -1)
	}

	return streak, nil
}

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

func (d *DiscordNotifier) SendMessage(content string) error {
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
		return fmt.Errorf("webhook failed: %d", resp.StatusCode)
	}
	return nil
}

type RealTimeProvider struct{}

func (RealTimeProvider) Now() time.Time {
	return time.Now()
}

// --- メイン関数 ---

func main() {
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config.json: %v", err)
	}
	var cfg service.Config
	if err := json.Unmarshal(configFile, &cfg); err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	if cfg.PollIntervalMinutes <= 0 {
		cfg.PollIntervalMinutes = 5
	}

	db, err := sql.Open("sqlite3", "streaks.db")
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	defer db.Close()

	initTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		atcoder_id TEXT PRIMARY KEY,
		last_streak INTEGER DEFAULT 0,
		last_checked_at TEXT
	);
	CREATE TABLE IF NOT EXISTS unique_ac (
		atcoder_id TEXT,
		problem_id TEXT,
		first_ac_epoch INTEGER,
		PRIMARY KEY (atcoder_id, problem_id)
	);
	CREATE TABLE IF NOT EXISTS daily_status (
		atcoder_id TEXT,
		date_jst TEXT,
		notified_success INTEGER DEFAULT 0,
		PRIMARY KEY (atcoder_id, date_jst)
	);`
	if _, err := db.Exec(initTableQuery); err != nil {
		log.Fatalf("Failed to init tables: %v", err)
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		jst = time.FixedZone("JST", 9*60*60)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	client := &RealAtCoderClient{client: httpClient}
	repo := &SQLiteRepository{db: db}
	notifier := &DiscordNotifier{webhookURL: cfg.WebhookURL, client: httpClient}
	timeProv := RealTimeProvider{}

	svc := service.NewWatchdogService(cfg, client, repo, notifier, timeProv, jst)

	c := cron.New(cron.WithLocation(jst))
	_, err = c.AddFunc(fmt.Sprintf("*/%d * * * *", cfg.PollIntervalMinutes), func() {
		svc.PollUniqueAC()
	})
	if err != nil {
		log.Fatalf("Failed to schedule polling: %v", err)
	}

	for _, n := range cfg.Notifications {
		parts := strings.Split(n.Time, ":")
		if len(parts) != 2 {
			continue
		}
		cronSpec := fmt.Sprintf("%s %s * * *", parts[1], parts[0])
		setting := n
		_, _ = c.AddFunc(cronSpec, func() {
			svc.HandleScheduledNotification(setting)
		})
	}
	c.Start()
	defer c.Stop()

	// テスト用エンドポイント（現在のステータスを即座に Discord へ送信）
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		go svc.CheckAndReportCurrentStatus()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"status report triggered"}`))
	})

	server := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		log.Println("Test API listening on :8080 (POST /test)")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Println("AtCoder Streak Watchdog started.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}