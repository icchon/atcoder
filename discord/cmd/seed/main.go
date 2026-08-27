package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Users []string `json:"users"`
}

type Submission struct {
	ID          int64  `json:"id"`
	EpochSecond int64  `json:"epoch_second"`
	ProblemID   string `json:"problem_id"`
	Result      string `json:"result"`
}

var jst *time.Location

func init() {
	var err error
	jst, err = time.LoadLocation("Asia/Tokyo")
	if err != nil {
		jst = time.FixedZone("JST", 9*60*60)
	}
}

func main() {
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config.json: %v", err)
	}
	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	db, err := sql.Open("sqlite3", "streaks.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	createTableQuery := `
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
	);`
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}

	for _, user := range config.Users {
		log.Printf("Seeding Unique AC for user: %s ...", user)
		if err := seedUserUniqueAC(db, client, user); err != nil {
			log.Printf("Error seeding %s: %v", user, err)
			continue
		}

		currentStreak, err := calculateCurrentStreak(db, user, time.Now())
		if err != nil {
			log.Printf("Error calculating current streak for %s: %v", user, err)
			currentStreak = 0
		}

		nowStr := time.Now().In(jst).Format(time.RFC3339)
		userUpsertQuery := `
		INSERT INTO users (atcoder_id, last_streak, last_checked_at)
		VALUES (?, ?, ?)
		ON CONFLICT(atcoder_id) DO UPDATE SET
			last_streak = excluded.last_streak,
			last_checked_at = excluded.last_checked_at;`
		if _, err := db.Exec(userUpsertQuery, user, currentStreak, nowStr); err != nil {
			log.Printf("Error updating users table for %s: %v", user, err)
		}

		log.Printf("User: %s | Current Streak: %d days", user, currentStreak)
	}

	log.Println("Seeding completed successfully.")
}

func seedUserUniqueAC(db *sql.DB, client *http.Client, user string) error {
	fromSecond := int64(0)
	totalFetched := 0

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO unique_ac (atcoder_id, problem_id, first_ac_epoch)
		VALUES (?, ?, ?)
		ON CONFLICT(atcoder_id, problem_id) DO UPDATE SET
			first_ac_epoch = MIN(unique_ac.first_ac_epoch, excluded.first_ac_epoch);
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for {
		url := fmt.Sprintf("https://kenkoooo.com/atcoder/atcoder-api/v3/user/submissions?user=%s&from_second=%d", user, fromSecond)
		resp, err := client.Get(url)
		if err != nil {
			return fmt.Errorf("API request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("bad status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		var subs []Submission
		if err := json.Unmarshal(body, &subs); err != nil {
			return err
		}

		if len(subs) == 0 {
			break
		}

		totalFetched += len(subs)
		lastEpoch := fromSecond

		for _, sub := range subs {
			if sub.EpochSecond > lastEpoch {
				lastEpoch = sub.EpochSecond
			}
			if sub.Result == "AC" {
				_, _ = stmt.Exec(user, sub.ProblemID, sub.EpochSecond)
			}
		}

		fromSecond = lastEpoch + 1

		if len(subs) < 500 {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return tx.Commit()
}

func calculateCurrentStreak(db *sql.DB, user string, now time.Time) (int, error) {
	rows, err := db.Query(`
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