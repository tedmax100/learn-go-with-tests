package v6

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestAfterFunc(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	calledCh := make(chan struct{}) // closed when AfterFunc is called
	context.AfterFunc(ctx, func() {
		close(calledCh)
	})

	var wg sync.WaitGroup
	wg.Add(1)

	// funcCalled reports whether the function was called.
	funcCalled := func() bool {
		select {
		case <-calledCh:
			return true
		case <-time.After(10 * time.Millisecond):
			return false
		}
	}

	// Assert that the AfterFunc has not been called.
	if funcCalled() {
		t.Fatalf("AfterFunc function called before context is canceled")
	}

	// Act
	cancel()

	// Assert that the AfterFunc has been called.
	if !funcCalled() {
		t.Fatalf("AfterFunc function not called after context is canceled")
	}

	// go test -run TestAfterFunc -v
	// 測試目標 context.AfterFunc 函式安排在 cancelable context 後，在自己的 goroutine 中呼叫函式。
	// 我們要檢查兩個條件：在 cancelable context 之前沒有呼叫函數，以及在 cancelable context 之後呼叫函數。
	// 此測試速度很慢：10 ms的時間並不長，但在多次測試中就會累積起來。
	// 此測試也很不穩定：在快速的電腦上，10 ms是很長的時間，但在共用和超載的 CI 系統上，停頓幾秒是很平常的事。
	// 我們可以讓測試變得不那麼鬆散，但代價是讓它變得更慢；我們也可以讓它變得不那麼慢，但代價是讓它變得更鬆散，但我們無法讓它既快又可靠。
}

func TestWithTimeout(t *testing.T) {
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("initial state, ctx.Err() = %v; want nil", err)
	}

	// 等待 context 被取消
	<-ctx.Done()

	// 確保 context 已經超時
	if err := ctx.Err(); err != context.DeadlineExceeded {
		t.Fatalf("after timeout, ctx.Err() = %v; want DeadlineExceeded", err)
	}

	// go test -run TestWithTimeout -v
}

func TestHTTPTimeout(t *testing.T) {
	// 建立一個測試用的 HTTP server
	server := createSlowServer()
	defer server.Close()

	t.Run("Context Timeout", func(t *testing.T) {
		// 建立一個有超時的 context
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// 建立請求
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/slow", nil)
		if err != nil {
			t.Fatalf("建立請求失敗: %v", err)
		}

		// 發送請求
		client := &http.Client{}
		_, err = client.Do(req)

		// 驗證是否如預期超時
		if err == nil {
			t.Error("預期應該發生超時錯誤，但沒有")
		}
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("預期 context.DeadlineExceeded，得到 %v", ctx.Err())
		}
	})

	t.Run("Client Timeout", func(t *testing.T) {
		client := &http.Client{
			Timeout: 100 * time.Millisecond,
		}

		// 發送請求
		_, err := client.Get(server.URL + "/slow")

		// 驗證是否如預期超時
		if err == nil {
			t.Error("預期應該發生超時錯誤，但沒有")
		}
		if !isTimeoutError(err) {
			t.Errorf("預期超時錯誤，得到 %v", err)
		}
	})

	t.Run("Transport Dial Timeout", func(t *testing.T) {
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 100 * time.Millisecond,
				}).DialContext,
			},
		}

		// 嘗試連接一個不存在的地址
		_, err := client.Get("http://10.255.255.1:8080")

		// 驗證是否如預期超時
		if err == nil {
			t.Error("預期應該發生連接超時錯誤，但沒有")
		}
		if !isTimeoutError(err) {
			t.Errorf("預期超時錯誤，得到 %v", err)
		}
	})

	t.Run("Transport Response Header Timeout", func(t *testing.T) {
		client := &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: 100 * time.Millisecond,
			},
		}

		// 發送請求到慢速伺服器
		_, err := client.Get(server.URL + "/slow-headers")

		// 驗證是否如預期超時
		if err == nil {
			t.Error("預期應該發生標頭超時錯誤，但沒有")
		}
		if !isTimeoutError(err) {
			t.Errorf("預期超時錯誤，得到 %v", err)
		}
	})

	// go test -run TestHTTPTimeout -v
}

// 建立一個慢速的測試伺服器
func createSlowServer() *httptest.Server {
	mux := http.NewServeMux()

	// 慢速回應的處理器
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprintln(w, "Finally responded!")
	})

	// 慢速標頭的處理器
	mux.HandleFunc("/slow-headers", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Headers finally sent!")
	})

	// 使用 httptest.NewServer 來創建測試服務器
	return httptest.NewServer(mux)
}

// 判斷錯誤是否為超時錯誤
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func TestSharedValue(t *testing.T) {
	// shared 是原子變數
	var shared atomic.Int64

	// 啟動一個 goroutine，來修改共用變數，它將 shared 設為 1，休眠 1 ms，然後設為 2。
	go func() {
		shared.Store(1)
		time.Sleep(1 * time.Microsecond)
		shared.Store(2)
	}()

	// Check the shared value after 5 microseconds
	time.Sleep(5 * time.Microsecond)
	if shared.Load() != 2 {
		t.Errorf("shared = %d, want 2", shared.Load())
	}

	// go test -run TestSharedValue -count=1000
	// Fail : shared = 1, want 2
	// Fail : shared = 0, want 2

	// 發生這種情況是因為測試不穩定。
	// 有時 goroutine 在檢查執行時尚未完成，甚至尚未啟動。結果取決於系統排程，以及 goroutine 被運行時間接收的速度。

	// time.Sleep 的精確度和排程器的行為可能差異很大。
	// 作業系統差異和系統負載等因素都會影響時序。這使得任何僅基於睡眠的同步策略都不可靠。

	// 受此類型鬆散性影響的真實系統包括背景清理、重試邏輯、基於時間的快取驅逐、心跳監控、分散式環境中的領導者選舉等。
	// 類似的測試取決於時序，也可能很花時間。想像一下，如果它必須等待 5 秒，而不是只有 5 微秒。
}

func TestTimingWithoutSynctest(t *testing.T) {
	start := time.Now().UTC()
	time.Sleep(5 * time.Second)
	t.Log(time.Since(start))

	// go test -run TestTimingWithoutSynctest -v
	// 您會發現輸出從來都不是準確的 5s 。
	// 相反，它可能看起來像 5.329s 、5.394s 或 5.456s 。這些變化來自於系統排程和時序解析的延遲。
	// 所以這樣的測試其實非常的困難。
}

// https://github.com/golang/go/blob/4bc3373c8e2cad24a779698477704306548949cb/src/testing/synctest/synctest.go#L5

func TestGoCacheEntryExpires(t *testing.T) {
	c := cache.New(5*time.Second, 10*time.Second)
	c.Set("foo", "bar", cache.DefaultExpiration)
	v, found := c.Get("foo")
	assert.True(t, found)
	assert.Equal(t, "bar", v)
	time.Sleep(5 * time.Second)
	v, found = c.Get("foo")
	assert.False(t, found)
	assert.Nil(t, v)
}
