package monitor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestTokenMonitor_v2(t *testing.T) {
	synctest.Run(func() {
		// Arrange
		notificationChan := make(chan string, 5)
		tm := NewTokenMonitor(notificationChan)
		tm.SetInterval(100 * time.Millisecond) // 設定較短的時間方便測試

		// 紀錄函數調用次數
		var checkFuncCalled int32
		var notificationsProcessed int32

		// 創建一個檢查點通道，用於確認檢查函數被調用
		checkDone := make(chan struct{}, 1)

		tm.SetCheckFunc(func(ctx context.Context) {
			atomic.AddInt32(&checkFuncCalled, 1)
			select {
			case checkDone <- struct{}{}:
			default:
			}
		})

		// 設定通知func 用於測試
		originalProcessFunc := tm.ProcessNotification
		tm.ProcessNotification = func(msg string) {
			if originalProcessFunc != nil {
				originalProcessFunc(msg)
			}
			atomic.AddInt32(&notificationsProcessed, 1)
		}

		// Act
		go tm.Run()
		defer tm.Stop()

		// 發送測試通知
		notificationChan <- "test message"

		// 等待檢查函數被調用或逾時
		select {
		case <-checkDone:
			// 檢查函數已被調用
		case <-time.After(200 * time.Millisecond):
			// 如果沒有被調用，synctest.Wait() 後會失敗
		}

		// 等待通知被處理和所有goroutine完成
		synctest.Wait()

		// Assert
		if atomic.LoadInt32(&checkFuncCalled) == 0 {
			t.Error("檢查函數未被調用")
		}

		if atomic.LoadInt32(&notificationsProcessed) != 1 {
			t.Errorf("通知處理次數不符合，期望1次，實際%d次", notificationsProcessed)
		}
	})

	// GOEXPERIMENT=synctest go test -race -run TestTokenMonitor_v2 -v
}

func TestTokenMonitor_NotificationProcessing_v2(t *testing.T) {
	synctest.Run(func() {
		// Arrange
		notificationChan := make(chan string, 5)
		tm := NewTokenMonitor(notificationChan)

		// 記錄處理的通知
		var processedNotifications []string
		var mu sync.Mutex
		processedCount := atomic.Int32{}

		// 處理完成的通知計數器
		expectedNotifications := 3

		// 通知處理完成的通道
		allProcessed := make(chan struct{})

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

			// 如果處理完了所有通知，發送信號
			if processedCount.Add(1) == int32(expectedNotifications) {
				close(allProcessed)
			}
		}

		// Act
		go tm.Run()
		defer tm.Stop()

		// 發送多個不同的通知
		notifications := []string{"notification1", "notification2", "notification3"}
		for _, msg := range notifications {
			notificationChan <- msg
		}

		// 等待所有通知被處理或逾時
		select {
		case <-allProcessed:
			// 所有通知已處理
		case <-time.After(500 * time.Millisecond):
			// 逾時，可能有通知未處理
		}

		// 等待所有goroutine完成
		synctest.Wait()

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
	})

	// GOEXPERIMENT=synctest go test -race -run TestTokenMonitor_NotificationProcessing_v2 -v
}

// 併發安全測試
func TestTokenMonitor_ConcurrencySafety_v2(t *testing.T) {
	// 測試1：多個goroutine同時發送通知
	t.Run("ConcurrentNotifications", func(t *testing.T) {
		synctest.Run(func() {
			notificationChan := make(chan string, 100) // 使用較大的緩衝區
			tm := NewTokenMonitor(notificationChan)

			var processedCount atomic.Int32

			const numGoroutines = 10
			const notificationsPerGoroutine = 10
			expectedTotal := numGoroutines * notificationsPerGoroutine

			// 用於等待所有通知被處理
			allProcessed := make(chan struct{})

			tm.ProcessNotification = func(msg string) {
				// 模擬處理時間
				time.Sleep(1 * time.Millisecond)

				count := processedCount.Add(1)
				if int(count) == expectedTotal {
					close(allProcessed)
				}
			}

			go tm.Run()
			defer tm.Stop()

			// 使用多個goroutine同時發送通知
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

			// 等待所有通知被處理或逾時
			select {
			case <-allProcessed:
				// 所有通知已處理
			case <-time.After(1 * time.Second):
				// 逾時，可能有通知未處理
			}

			// 等待所有goroutine完成
			synctest.Wait()

			// 檢查是否所有通知都被處理
			expected := int32(expectedTotal)
			if processedCount.Load() != expected {
				t.Errorf("通知處理數量不符，預期%d，實際%d", expected, processedCount.Load())
			}
		})
	})

	// 測試2：檢查函數執行時間較長
	t.Run("LongRunningCheckFunction", func(t *testing.T) {
		synctest.Run(func() {
			notificationChan := make(chan string, 5)
			tm := NewTokenMonitor(notificationChan)
			tm.SetInterval(50 * time.Millisecond)

			var checkFuncRunning atomic.Int32
			var maxConcurrentChecks atomic.Int32

			// 用於檢測是否至少執行了2次檢查
			checksExecuted := atomic.Int32{}
			checksDone := make(chan struct{})

			tm.SetCheckFunc(func(ctx context.Context) {
				// 記錄當前正在執行的檢查函數數量
				current := checkFuncRunning.Add(1)
				if current > maxConcurrentChecks.Load() {
					maxConcurrentChecks.Store(current)
				}

				// 模擬長時間運行的檢查函數
				time.Sleep(100 * time.Millisecond)

				checkFuncRunning.Add(-1)

				// 計數執行完成的檢查函數
				count := checksExecuted.Add(1)
				if count >= 2 {
					select {
					case checksDone <- struct{}{}:
					default:
					}
				}
			})

			go tm.Run()
			defer tm.Stop()

			// 等待至少執行完成2次檢查函數或逾時
			select {
			case <-checksDone:
				// 成功執行了多次
			case <-time.After(500 * time.Millisecond):
				// 逾時
			}

			// 確保所有goroutine完成
			synctest.Wait()

			// 檢查是否有多個檢查函數同時運行
			if maxConcurrentChecks.Load() <= 1 {
				t.Error("未檢測到併發執行的檢查函數")
			}
		})
	})

	// 測試3：在檢查函數執行過程中修改間隔時間
	t.Run("ChangeIntervalDuringCheck", func(t *testing.T) {
		synctest.Run(func() {
			notificationChan := make(chan string, 5)
			tm := NewTokenMonitor(notificationChan)
			tm.SetInterval(100 * time.Millisecond)

			var checkStarted atomic.Bool
			var intervalChanged atomic.Bool
			var checkAfterChange atomic.Bool

			// 檢查開始執行的信號
			checkStartedCh := make(chan struct{})
			// 檢查在間隔修改後執行的信號
			checkAfterChangeCh := make(chan struct{})

			tm.SetCheckFunc(func(ctx context.Context) {
				if !checkStarted.Swap(true) {
					// 第一次執行，通知測試
					close(checkStartedCh)
					// 長時間運行的檢查函數
					time.Sleep(150 * time.Millisecond)
					return
				}

				if intervalChanged.Load() {
					checkAfterChange.Store(true)
					close(checkAfterChangeCh)
				}
			})

			go tm.Run()
			defer tm.Stop()

			// 等待檢查函數開始執行
			<-checkStartedCh

			// 在檢查函數執行過程中修改間隔時間
			tm.SetInterval(50 * time.Millisecond)
			intervalChanged.Store(true)

			// 等待修改間隔後的檢查函數被調用或逾時
			select {
			case <-checkAfterChangeCh:
				// 成功，有新的檢查函數被調用
			case <-time.After(300 * time.Millisecond):
				t.Error("修改間隔後未有新的檢查函數被調用")
			}

			// 確保所有goroutine完成
			synctest.Wait()
		})
	})

	// 測試4：在檢查函數執行過程中停止服務
	t.Run("StopDuringCheck", func(t *testing.T) {
		synctest.Run(func() {
			notificationChan := make(chan string, 5)
			tm := NewTokenMonitor(notificationChan)

			var checkStarted atomic.Bool
			var checkCompleted atomic.Bool

			// 檢查開始執行的信號
			checkStartedCh := make(chan struct{})
			// 檢查完成的信號
			checkCompletedCh := make(chan struct{})

			tm.SetCheckFunc(func(ctx context.Context) {
				if !checkStarted.Swap(true) {
					close(checkStartedCh)

					// 檢查是否在函數執行過程中context被取消
					select {
					case <-time.After(200 * time.Millisecond):
						checkCompleted.Store(true)
						close(checkCompletedCh)
					case <-ctx.Done():
						// context被取消，不標記為完成
						return
					}
				}
			})

			go tm.Run()

			// 等待檢查函數開始執行
			<-checkStartedCh

			// 在檢查函數執行過程中停止服務
			tm.Stop()

			// 等待一段時間看是否會完成
			select {
			case <-checkCompletedCh:
				t.Error("停止服務後檢查函數仍完成執行")
			case <-time.After(300 * time.Millisecond):
				// 正常情況，檢查函數未完成
			}

			// 確保所有goroutine完成
			synctest.Wait()

			// 檢查函數應該因為context取消而未完成
			if checkCompleted.Load() {
				t.Error("停止服務後檢查函數仍完成執行")
			}
		})
	})
	// GOEXPERIMENT=synctest go test -race -run TestTokenMonitor_ConcurrencySafety_v2 -v
}
