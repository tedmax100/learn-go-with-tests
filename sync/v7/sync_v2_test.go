package v7

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/patrickmn/go-cache"
)

func TestAfterFunc(t *testing.T) {
	synctest.Run(func() {
		ctx, cancel := context.WithCancel(context.Background())

		funcCalled := false
		context.AfterFunc(ctx, func() {
			funcCalled = true
		})

		synctest.Wait()
		if funcCalled {
			t.Fatalf("AfterFunc function called before context is canceled")
		}

		cancel()

		synctest.Wait()
		// 如果把這 Wait() 給註解，go test race detector 會發現該測試存在 data race。
		if !funcCalled {
			t.Fatalf("AfterFunc function not called after context is canceled")
		}
	})

	// GOEXPERIMENT=synctest go test -run TestAfterFunc -v

	// 測試也更簡單了：我們用一個 boolean 取代了 calledCh channel。
	// 之前我們需要使用 channel 來避免測試 goroutine 與 AfterFunc goroutine 之間的 data racing，
	// 但現在 Wait 函式提供了同步功能。

	// go test race detector 依然能使用。
	// GOEXPERIMENT=synctest go test -race -run TestAfterFunc -v
}

func TestTimingWithSynctest(t *testing.T) {
	synctest.Run(func() {
		start := time.Now().UTC()
		time.Sleep(5 * time.Second)
		t.Log(time.Since(start))
	})

	// GOEXPERIMENT=synctest go test -run TestTimingWithSynctest -v
	// 使用 synctest 時，時間完全受控。
	// time.Sleep 內的 synctest 會立即返回。測試實際上不會等待 5 秒。這會使測試執行得更快，同時仍然精確。
}

func createTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		// 在 synctest 環境中使用 Sleep
		time.Sleep(2 * time.Second)
		fmt.Fprintln(w, "Finally responded!")
	})

	mux.HandleFunc("/slow-headers", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Headers finally sent!")
	})

	return httptest.NewServer(mux)
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func TestSharedValue(t *testing.T) {
	synctest.Run(func() {
		var shared atomic.Int64
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
	})

	// GOEXPERIMENT=synctest go test -run TestSharedValue -count=1000

	// 5 ms是模擬而非真實的。當程式碼執行時，時間實際上是凍結的，synctest 會管理其process。
	// 換句話說，邏輯並不依賴實際時間，而是取決於確定的執行順序。
}

func TestConcurrentNetworkRequests(t *testing.T) {
	synctest.Run(func() {
		// 模擬多個客戶端並發訪問伺服器
		for i := 0; i < 100; i++ {
			go func() {
				resp, err := http.Get("http://example.com")
				if err != nil {
					t.Errorf("network request failed: %v", err)
				}
				resp.Body.Close()
			}()
		}

		// 使用 synctest.Wait 等待所有並發操作完成
		synctest.Wait()
	})
}

func TestGoCacheEntryExpiresWithSynctest(t *testing.T) {
	c := cache.New(2*time.Second, 5*time.Second)
	synctest.Run(func() {
		c.Set("foo", "bar", cache.DefaultExpiration)
		// Get an entry from the cache.
		if got, exist := c.Get("foo"); !exist && got != "bar" {
			t.Errorf("c.Get(k) = %v, want %v", got, "bar")
		}

		// Verify that we get the same entry when accessing it before the expiry.
		time.Sleep(1 * time.Second)
		if got, exist := c.Get("foo"); !exist && got != "bar" {
			t.Errorf("c.Get(k) = %v, want %v", got, "bar")
		}
		// Wait for the entry to expire and verify that we now get a new one.
		time.Sleep(3 * time.Second)
		if got, exist := c.Get("foo"); exist {
			t.Errorf("c.Get(k) = %v, want %v", got, nil)
		}
	})

	// GOEXPERIMENT=synctest go test -run TestGoCacheEntryExpiresWithSynctest -v
}

func TestAA(t *testing.T) {
	synctest.Run(func() {
		ctx := context.Background()

		ctx, cancel := context.WithCancel(ctx)

		var hits atomic.Int32
		go func() {
			tick := time.NewTicker(time.Millisecond)
			// no defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					hits.Add(1)
				}
			}
		}()

		time.Sleep(3 * time.Millisecond)
		cancel()

		got := int(hits.Load())
		if want := 3; got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
	// GOEXPERIMENT=synctest go test -run TestAA -v
}
