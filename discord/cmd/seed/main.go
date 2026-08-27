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

func main() {
	// 設定読み込み
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config.json: %v", err)
	}
	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	// SQLite 初期化
	db, err := sql.Open("sqlite3", "streaks.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS unique_ac (
		atcoder_id TEXT,
		problem_id TEXT,
		first_ac_epoch INTEGER,
		PRIMARY KEY (atcoder_id, problem_id)
	);`
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	for _, user := range config.Users {
		log.Printf("Seeding Unique AC for user: %s ...", user)
		if err := seedUserUniqueAC(db, user); err != nil {
			log.Printf("Error seeding %s: %v", user, err)
		}
	}

	log.Println("Seeding completed successfully.")
}

func seedUserUniqueAC(db *sql.DB, user string) error {
	client := http.Client{Timeout: 15 * time.Second}
	fromSecond := int64(0)
	totalFetched := 0
	uniqueCount := 0

	// トランザクション開始
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
				res, err := stmt.Exec(user, sub.ProblemID, sub.EpochSecond)
				if err == nil {
					if rows, _ := res.RowsAffected(); rows > 0 {
						uniqueCount++
					}
				}
			}
		}

		// 次のページネーションの開始秒
		fromSecond = lastEpoch + 1

		// 取得件数が 500 件未満なら全件取得完了
		if len(subs) < 500 {
			break
		}

		// API Rate limit への配慮
		time.Sleep(1 * time.Second)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 登録済みの Unique AC 数をカウント
	var totalUnique int
	_ = db.QueryRow("SELECT COUNT(*) FROM unique_ac WHERE atcoder_id = ?", user).Scan(&totalUnique)

	log.Printf("User: %s | Fetched Submissions: %d | Total Unique AC: %d", user, totalFetched, totalUnique)
	return nil
}
