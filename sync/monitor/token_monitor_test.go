package monitor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenMonitor(t *testing.T) {
	// Arrange
	notificationChan := make(chan string, 5)

	tm := NewTokenMonitor(notificationChan)
	tm.SetInterval(100 * time.Millisecond) // 設定較短的時間方便測試

	// 紀錄函数調用次數
	var checkFuncCalled atomic.Int32
	var notificationsProcessed atomic.Int32

	tm.SetCheckFunc(func(ctx context.Context) {
		checkFuncCalled.Add(1)
	})

	// 設定通知func 用於測試
	originalProcessFunc := tm.ProcessNotification
	tm.ProcessNotification = func(msg string) {
		originalProcessFunc(msg)
		notificationsProcessed.Add(1)
	}

	// Act
	go tm.Run()

	notificationChan <- "test message"

	// 困難點1: 需要使用 Sleep 等待非同步操作完成
	// 這種方式不可靠，可能導致測試不穩定
	time.Sleep(300 * time.Millisecond)

	// 困難點2: 無法準確知道何時檢查函數被調用了
	if checkFuncCalled.Load() == 0 {
		t.Error("檢查函數未被調用")
	}

	// 困難點3: 不好確認調用次數
	// 由于時間因素，有可能調用1次或多次
	if notificationsProcessed.Load() != 1 {
		t.Errorf("通知處理次數不符合，期望1次，實際%d次", notificationsProcessed.Load())
	}

	// 困難點4: 不好確保所有goroutine都執行完成
	// 可能在某些操作仍在進行時就停止了測試

	tm.Stop()
}

// 場景2：通知處理測試
func TestTokenMonitor_NotificationProcessing(t *testing.T) {
	// Arrange
	notificationChan := make(chan string, 5)
	tm := NewTokenMonitor(notificationChan)

	// 記錄處理的通知
	var processedNotifications []string
	var mu sync.Mutex

	tm.ProcessNotification = func(msg string) {
		mu.Lock()
		processedNotifications = append(processedNotifications, msg)
		mu.Unlock()

		// 模擬處理時間不同
		sleepTime := 10 * time.Millisecond
		if msg == "notification2" {
			sleepTime = 5 * time.Millisecond
		}
		time.Sleep(sleepTime)
	}

	// Act
	go tm.Run()

	// 發送多個不同的通知
	notifications := []string{"notification1", "notification2", "notification3"}
	for _, msg := range notifications {
		notificationChan <- msg
	}

	// 等待足夠的時間讓所有通知被處理
	time.Sleep(100 * time.Millisecond)

	// 停止監控
	tm.Stop()

	// Assert
	mu.Lock()
	defer mu.Unlock()

	// 驗證所有通知都被處理
	if len(processedNotifications) != len(notifications) {
		t.Errorf("通知處理數量不符，預期%d，實際%d", len(notifications), len(processedNotifications))
	}

	// 檢查是否所有通知都被處理
	notificationMap := make(map[string]bool)
	for _, msg := range processedNotifications {
		notificationMap[msg] = true
	}

	for _, msg := range notifications {
		if !notificationMap[msg] {
			t.Errorf("通知 '%s' 未被處理", msg)
		}
	}

	// 注意：由於使用 goroutine 處理通知，無法保證處理順序與發送順序一致
	// 這是傳統測試的限制之一
}

// 場景3：併發安全測試
func TestTokenMonitor_ConcurrencySafety(t *testing.T) {
	// Arrange
	notificationChan := make(chan string, 100) // 使用較大的緩衝區
	tm := NewTokenMonitor(notificationChan)

	// 測試1：多個goroutine同時發送通知
	t.Run("ConcurrentNotifications", func(t *testing.T) {
		var processedCount atomic.Int32

		tm.ProcessNotification = func(msg string) {
			processedCount.Add(1)
			// 模擬處理時間
			time.Sleep(1 * time.Millisecond)
		}

		go tm.Run()

		// 使用多個goroutine同時發送通知
		const numGoroutines = 10
		const notificationsPerGoroutine = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < notificationsPerGoroutine; j++ {
					notificationChan <- fmt.Sprintf("notification-%d-%d", id, j)
					// 小延遲，避免所有消息立即發送
					time.Sleep(time.Millisecond)
				}
			}(i)
		}

		// 等待所有goroutine完成發送
		wg.Wait()

		// 等待足夠的時間讓所有通知被處理
		time.Sleep(200 * time.Millisecond)

		// 檢查是否所有通知都被處理
		expected := int32(numGoroutines * notificationsPerGoroutine)
		if processedCount.Load() != expected {
			t.Errorf("通知處理數量不符，預期%d，實際%d", expected, processedCount.Load())
		}

		tm.Stop()
	})

	// 測試2：檢查函數執行時間較長
	t.Run("LongRunningCheckFunction", func(t *testing.T) {
		notificationChan = make(chan string, 5)
		tm = NewTokenMonitor(notificationChan)
		tm.SetInterval(50 * time.Millisecond)

		var checkFuncRunning atomic.Int32
		var maxConcurrentChecks atomic.Int32

		tm.SetCheckFunc(func(ctx context.Context) {
			// 記錄當前正在執行的檢查函數數量
			current := checkFuncRunning.Add(1)
			if current > maxConcurrentChecks.Load() {
				maxConcurrentChecks.Store(current)
			}

			// 模擬長時間運行的檢查函數
			time.Sleep(100 * time.Millisecond)

			checkFuncRunning.Add(-1)
		})

		go tm.Run()

		// 等待足夠的時間讓多個檢查函數重疊執行
		time.Sleep(250 * time.Millisecond)

		tm.Stop()

		// 檢查是否有多個檢查函數同時運行
		if maxConcurrentChecks.Load() <= 1 {
			t.Error("未檢測到併發執行的檢查函數")
		}
	})

	// 測試3：在檢查函數執行過程中修改間隔時間
	t.Run("ChangeIntervalDuringCheck", func(t *testing.T) {
		notificationChan = make(chan string, 5)
		tm = NewTokenMonitor(notificationChan)
		tm.SetInterval(100 * time.Millisecond)

		var checkStarted atomic.Bool
		var intervalChanged atomic.Bool
		var checkAfterChange atomic.Bool

		tm.SetCheckFunc(func(ctx context.Context) {
			if intervalChanged.Load() {
				checkAfterChange.Store(true)
				return
			}

			checkStarted.Store(true)
			// 長時間運行的檢查函數
			time.Sleep(150 * time.Millisecond)
		})

		go tm.Run()

		// 等待檢查函數開始執行
		for !checkStarted.Load() {
			time.Sleep(10 * time.Millisecond)
		}

		// 在檢查函數執行過程中修改間隔時間
		tm.SetInterval(50 * time.Millisecond)
		intervalChanged.Store(true)

		// 等待足夠的時間讓新間隔生效
		time.Sleep(200 * time.Millisecond)

		tm.Stop()

		// 檢查修改間隔後是否有新的檢查函數被調用
		if !checkAfterChange.Load() {
			t.Error("修改間隔後未有新的檢查函數被調用")
		}
	})

	// 測試4：在檢查函數執行過程中停止服務
	t.Run("StopDuringCheck", func(t *testing.T) {
		notificationChan = make(chan string, 5)
		tm = NewTokenMonitor(notificationChan)

		var checkStarted atomic.Bool
		var checkCompleted atomic.Bool

		tm.SetCheckFunc(func(ctx context.Context) {
			checkStarted.Store(true)

			// 檢查是否在函數執行過程中context被取消
			select {
			case <-time.After(200 * time.Millisecond):
				checkCompleted.Store(true)
			case <-ctx.Done():
				// context被取消，不標記為完成
				return
			}
		})

		go tm.Run()

		// 等待檢查函數開始執行
		for !checkStarted.Load() {
			time.Sleep(10 * time.Millisecond)
		}

		// 在檢查函數執行過程中停止服務
		tm.Stop()

		// 等待一段時間
		time.Sleep(250 * time.Millisecond)

		// 檢查函數應該因為context取消而未完成
		if checkCompleted.Load() {
			t.Error("停止服務後檢查函數仍完成執行")
		}
	})
}
