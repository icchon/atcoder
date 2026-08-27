package service

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type WatchdogService struct {
	cfg      Config
	client   AtCoderClient
	repo     Repository
	notifier Notifier
	timeProv TimeProvider
	jst      *time.Location
	mu       sync.Mutex
}

func NewWatchdogService(
	cfg Config,
	client AtCoderClient,
	repo Repository,
	notifier Notifier,
	timeProv TimeProvider,
	jst *time.Location,
) *WatchdogService {
	return &WatchdogService{
		cfg:      cfg,
		client:   client,
		repo:     repo,
		notifier: notifier,
		timeProv: timeProv,
		jst:      jst,
	}
}

func (s *WatchdogService) getTodayJST() (string, int64) {
	now := s.timeProv.Now().In(s.jst)
	dateStr := now.Format("2006-01-02")
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.jst).Unix()
	return dateStr, startOfDay
}

// 5分ごとの Unique AC 検知処理
func (s *WatchdogService) PollUniqueAC() {
	s.mu.Lock()
	defer s.mu.Unlock()

	dateStr, startOfDayEpoch := s.getTodayJST()

	for _, user := range s.cfg.Users {
		notified, err := s.repo.IsSuccessNotified(user, dateStr)
		if err != nil {
			log.Printf("Error checking notification status for %s: %v", user, err)
			continue
		}
		if notified {
			continue
		}

		subs, err := s.client.FetchSubmissions(user, startOfDayEpoch)
		if err != nil {
			log.Printf("Error fetching submissions for %s: %v", user, err)
			continue
		}

		var newSolved []string
		for _, sub := range subs {
			if sub.Result != "AC" {
				continue
			}

			solved, err := s.repo.IsProblemSolved(user, sub.ProblemID)
			if err != nil {
				log.Printf("Error checking problem %s for %s: %v", sub.ProblemID, user, err)
				continue
			}

			if !solved {
				if err := s.repo.SaveUniqueAC(user, sub.ProblemID, sub.EpochSecond); err == nil {
					newSolved = append(newSolved, sub.ProblemID)
				}
			}
		}

		if len(newSolved) > 0 {
			streak, _ := s.repo.GetCurrentStreak(user, s.timeProv.Now(), s.jst)
			msg := fmt.Sprintf(" **`%s` が本日の Unique AC を達成しました！**\n"+
				"解いた問題: `%v`\n"+
				"現在の記録: **%d** 日連続 \n"+
				"本日のノルマ達成！お疲れ様でした！", user, newSolved, streak)

			_ = s.notifier.SendMessage(msg)
			_ = s.repo.SetSuccessNotified(user, dateStr)
		}
	}
}

// 定期催促・最終警告通知
func (s *WatchdogService) HandleScheduledNotification(n NotificationSetting) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dateStr, startOfDayEpoch := s.getTodayJST()
	now := s.timeProv.Now().In(s.jst)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, s.jst)
	remaining := endOfDay.Sub(now).Round(time.Minute)

	for _, user := range s.cfg.Users {
		if s.isTodaySolved(user, dateStr, startOfDayEpoch) {
			continue
		}

		streak, _ := s.repo.GetCurrentStreak(user, s.timeProv.Now(), s.jst)
		var msg string

		switch n.Type {
		case "final_warning":
			msg = fmt.Sprintf(" **【最終警告】`%s` の Streak 終了まで残り 約 %d 分です！** \n"+
				"現在の Streak: **%d** 日\n"+
				"まだ今日の Unique AC が確認できていません！\n"+
				"今すぐ提出してください！  https://kenkoooo.com/atcoder/#/table/%s",
				user, int(remaining.Minutes()), streak, user)
		default: // reminder
			msg = fmt.Sprintf(" **【%s リマインド】`%s` は今日まだ Unique AC がありません！**\n"+
				"現在の Streak: **%d** 日\n"+
				"日付変更まで残り: **約 %d 時間 %d 分**\n"+
				"Streak を維持するために 1 問解きましょう！  https://kenkoooo.com/atcoder/#/table/%s",
				n.Time, user, streak, int(remaining.Hours()), int(remaining.Minutes())%60, user)
		}

		_ = s.notifier.SendMessage(msg)
	}
}

// 現在のステータス（達成 / 未達成）を判定して即時通知
func (s *WatchdogService) CheckAndReportCurrentStatus() {
	s.mu.Lock()
	defer s.mu.Unlock()

	dateStr, startOfDayEpoch := s.getTodayJST()
	now := s.timeProv.Now().In(s.jst)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, s.jst)
	remaining := endOfDay.Sub(now).Round(time.Minute)

	for _, user := range s.cfg.Users {
		// 未登録 AC があれば同期
		subs, err := s.client.FetchSubmissions(user, startOfDayEpoch)
		if err == nil {
			for _, sub := range subs {
				if sub.Result == "AC" {
					solved, _ := s.repo.IsProblemSolved(user, sub.ProblemID)
					if !solved {
						_ = s.repo.SaveUniqueAC(user, sub.ProblemID, sub.EpochSecond)
					}
				}
			}
		}

		isSolved := s.isTodaySolved(user, dateStr, startOfDayEpoch)
		streak, _ := s.repo.GetCurrentStreak(user, s.timeProv.Now(), s.jst)

		var msg string
		if isSolved {
			msg = fmt.Sprintf(" **`%s` は本日の Unique AC を達成済みです！**\n"+
				"現在の Streak: **%d** 日連続 \n"+
				"本日のノルマは完了しています。", user, streak)
		} else {
			msg = fmt.Sprintf(" **`%s` は今日まだ Unique AC がありません！**\n"+
				"現在の Streak: **%d** 日\n"+
				"日付変更まで残り: **約 %d 時間 %d 分**\n"+
				"Streak を維持するために 1 問解きましょう！  https://kenkoooo.com/atcoder/#/table/%s",
				user, streak, int(remaining.Hours()), int(remaining.Minutes())%60, user)
		}

		_ = s.notifier.SendMessage(msg)
	}
}

func (s *WatchdogService) isTodaySolved(user, dateStr string, startOfDayEpoch int64) bool {
	notified, err := s.repo.IsSuccessNotified(user, dateStr)
	if err == nil && notified {
		return true
	}

	hasAC, err := s.repo.HasUniqueACSince(user, startOfDayEpoch)
	return err == nil && hasAC
}