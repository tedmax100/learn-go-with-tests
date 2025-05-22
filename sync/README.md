# Sync 同步操作

---

# 對 Counter 做測試
Counter 是嚴格遞增的，因此只會有兩個 api `inc())` 與 `value()`。 
我們首先要測試只有一個 goroutine 在操作 Counter時的測試場景，正常的話我們調用幾次 inc 那 value 也會是對應的調用次數。

以下測試就是調用了3次，因此我們會希望 value 返回的計數也會是 `3`。
```go
t.Run("incrementing the counter 3 times leaves it at 3", func(t *testing.T) {
		counter := Counter{}
		counter.Inc()
		counter.Inc()
		counter.Inc()

		assertCounter(t, counter, 3)
	})
```
那這時就能設計 counter 的實做了。實做如下很簡單。
```go
// Counter will increment a number.
type Counter struct {
	value int
}

// Inc the count.
func (c *Counter) Inc() {
	c.value++
}

// Value returns the current count.
func (c *Counter) Value() int {
	return c.value
}
```

在運行一次測試會發現就過了。
```bash
go test -v -run TestCounter/incrementing ./v1
=== RUN   TestCounter
=== RUN   TestCounter/incrementing_the_counter_3_times_leaves_it_at_3
--- PASS: TestCounter (0.00s)
    --- PASS: TestCounter/incrementing_the_counter_3_times_leaves_it_at_3 (0.00s)
PASS
ok      github.com/quii/learn-go-with-tests/sync/v1     0.002s
```

# 併發測試
在 Go 中能使用 `goroutine` 來對 counter 進行併發操作。因此能用 goroutine 來進行多次inc()，然後透過value()取值來檢測。 

以下這段簡單說就是起了 `1000` 個 goroutine去執行 counter 的inc()。然後執行完成就對 waitgroup 說自己這gorotine 結束了。然後所有 goroutine 都跑完了就檢查 counter 的值是否為 `1000`。
```go
t.Run("it runs safely concurrently", func(t *testing.T) {
    wantedCount := 1000
    counter := &Counter{}

    var wg sync.WaitGroup
    wg.Add(wantedCount)

    for i := 0; i < wantedCount; i++ {
        go func() {
            counter.Inc()
            wg.Done()
        }()
    }
    wg.Wait()

    assertCounter(t, counter, wantedCount)
})
```

> waitgroup 本身由 `counter`、`waiters`和`semaphore`組成。Add 可以給正數或負數，這裡會直接對 counter 加減，如果 counter 為 0 表示goroutine 所有任務已經完成，就會去喚醒所有被阻塞在這等待的 goroutine （waiter）起床。`semaphore`目的是用來阻塞跟喚醒goroutine用的。
> 所以在這例子中，`wg.Add(wantedCount)` 先替 counter 加了 1000，`wg.Done()`時就開始替counter -1。而 測試函數本身這goroutine 則被迫阻塞在等待 `wg.Wait()`來喚醒它。

執行看看測試會發現第二個併發的測試出錯了，不是想像中的`1000`。
```bash
go test -v -run TestCounter ./v2
=== RUN   TestCounter
=== RUN   TestCounter/incrementing_the_counter_3_times_leaves_it_at_3
=== RUN   TestCounter/it_runs_safely_concurrently
    sync_test.go:34: got 985, want 1000
--- FAIL: TestCounter (0.00s)
    --- PASS: TestCounter/incrementing_the_counter_3_times_leaves_it_at_3 (0.00s)
    --- FAIL: TestCounter/it_runs_safely_concurrently (0.00s)
FAIL
FAIL    github.com/quii/learn-go-with-tests/sync/v2     0.003s
FAIL
```

## 為什麼會失敗？
- counter.Inc() 實現是 c.value++，這個操作在底層不是原子性的。
- 多個 goroutine 同時執行 c.value++ 會產生race condition：
  - c.value++ 實際是「`讀取` -> `加 1` -> `寫回`」三個步驟。
  - 如果兩個 goroutine 同時`讀取到相同的值`，然後都加 1 再寫回，結果只增加了 `1` 而不是 `2`。

因此，最終計數值會比預期小。

## 解法1 加互斥鎖（mutual exclusion）
在 Inc() 一開始就上鎖，完成就解鎖。來讓「`讀取` -> `加 1` -> `寫回`」三個步驟同時間只能一個goroutine在進行。

```go
import "sync"

// Counter will increment a number.
type Counter struct {
	mu    sync.Mutex
	value int
}

// NewCounter returns a new Counter.
func NewCounter() *Counter {
	return &Counter{}
}

// Inc the count.
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

// Value returns the current count.
func (c *Counter) Value() int {
	return c.value
}
```

再執行一次測試，會發現就過了
```bash
go test -v -run TestCounter ./v2
=== RUN   TestCounter
=== RUN   TestCounter/incrementing_the_counter_3_times_leaves_it_at_3
=== RUN   TestCounter/it_runs_safely_concurrently
--- PASS: TestCounter (0.00s)
    --- PASS: TestCounter/incrementing_the_counter_3_times_leaves_it_at_3 (0.00s)
    --- PASS: TestCounter/it_runs_safely_concurrently (0.00s)
PASS
ok      github.com/quii/learn-go-with-tests/sync/v2     0.003s
```

## 解法2 原子操作
Go 的 sync 標準庫提了了很多基礎類型的原子操作。這裡我們能使用`atomic.AddInt64()`與`atomic.LoadInt64()`來處理。

> 原子操作（Atomic Operation）就是那3步驟是不可被別的gorotine給執行過程中不會被中斷或干擾。
> 也就是3個操作要麼全部完成，要麼完全不做（不會停在中間狀態）。也不會和其他原子操作互相干擾，避免race condition。
> atomic 保證這個操作在多核、多執行緒環境中看起來就像是「單執行緒」在執行這個加法——也就是說，這個操作是「不可中斷」且「線性化」的（linearizable）。
> 換句話說，雖然實際上有很多 goroutine 同時執行，但在操作這個特定變數時，CPU 會確保：
>   1. 每次只有一個 goroutine 的 AddInt64 操作能成功執行。
>   2. 其他 goroutine 必須排隊等待前一個操作完成，才能執行下一個。
> 這種排隊和互斥是由 CPU 的原子指令和快取一致性協議（cache coherence protocol）硬體層面保證的。

```go
package v2

import (
	"sync/atomic"
)

type ICounter interface {
	Inc()
	Value() int64
}

// Counter will increment a number safely in concurrent environment.
type AtomicCounter struct {
	value int64
}

// Inc increments the counter atomically.
func (c *AtomicCounter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Value returns the current count atomically.
func (c *AtomicCounter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}
```
加入一個新的測試對象是`AtomicCounter`，但跑一樣的測試方法。
```go
t.Run("it runs safely concurrently by Atomic", func(t *testing.T) {
    wantedCount := 1000
    counter := &AtomicCounter{}

    var wg sync.WaitGroup
    wg.Add(wantedCount)

    for i := 0; i < wantedCount; i++ {
        go func() {
            counter.Inc()
            wg.Done()
        }()
    }
    wg.Wait()

    assertCounter(t, counter, wantedCount)
})
```

執行看看測試，3個測試全過。
```bash
go test -v -run TestCounter ./v2
=== RUN   TestCounter
=== RUN   TestCounter/incrementing_the_counter_3_times_leaves_it_at_3
=== RUN   TestCounter/it_runs_safely_concurrently
=== RUN   TestCounter/it_runs_safely_concurrently_by_Atomic
--- PASS: TestCounter (0.00s)
    --- PASS: TestCounter/incrementing_the_counter_3_times_leaves_it_at_3 (0.00s)
    --- PASS: TestCounter/it_runs_safely_concurrently (0.00s)
    --- PASS: TestCounter/it_runs_safely_concurrently_by_Atomic (0.00s)
PASS
ok      github.com/quii/learn-go-with-tests/sync/v2     0.004s
```

# Bonus
## Go test race 檢測器

```go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	start := time.Now()
	var t *time.Timer
	t = time.AfterFunc(randomDuration(), func() {
		fmt.Println(time.Now().Sub(start))
		t.Reset(randomDuration())
	})

	time.Sleep(5 * time.Second)
}

func randomDuration() time.Duration {
	return time.Duration(rand.Int63n(1e9))
}
```

這段其實也隱藏著 race condition 問題。
Go 身為 cloud native 常用語言，提供了 Race detector 的工具。
我們可以 `go test -race ./...` 或者 `go run -race xxx.go` 或是 `go build -race xxx` 只要加入 `-race`就能啟用 race 檢測。

```bash
go run -race main.go
==================
WARNING: DATA RACE
Read at 0x00c00005c040 by goroutine 8:
  main.main.func1()
      /home/nathan/Project/learn-go-with-tests/sync/v3/main.go:14 +0xd3

Previous write at 0x00c00005c040 by main goroutine:
  main.main()
      /home/nathan/Project/learn-go-with-tests/sync/v3/main.go:12 +0x159

Goroutine 8 (running) created at:
  time.goFunc()
      /usr/local/go/src/time/sleep.go:215 +0x44
```

來移到單元測試中，第一個測試案例是原本main.go會發生race condition的版本。第二個是用mutex lock 來修復問題。第三個則是利用recursive function 來修復。

```go
package main_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// TestTimerRaceCondition
func TestTimerRaceCondition(t *testing.T) {
	start := time.Now()
	var timer *time.Timer

	timer = time.AfterFunc(randomDuration(), func() {
		fmt.Println(time.Now().Sub(start))
		timer.Reset(randomDuration())
	})

	time.Sleep(1 * time.Second)
}

// TestTimerWithMutex
func TestTimerWithMutex(t *testing.T) {
	start := time.Now()
	var timer *time.Timer
	var mu sync.Mutex

	mu.Lock()
	timer = time.AfterFunc(randomDuration(), func() {
		mu.Lock()
		fmt.Println(time.Now().Sub(start))
		timer.Reset(randomDuration())
		mu.Unlock()
	})
	mu.Unlock()

	time.Sleep(1 * time.Second)
}

// TestTimerWithRecursiveFunc
func TestTimerWithRecursiveFunc(t *testing.T) {
	start := time.Now()

	var scheduleNext func()
	scheduleNext = func() {
		fmt.Println(time.Now().Sub(start))
		time.AfterFunc(randomDuration(), scheduleNext)
	}

	time.AfterFunc(randomDuration(), scheduleNext)

	time.Sleep(1 * time.Second)
}

func randomDuration() time.Duration {
	return time.Duration(rand.Int63n(1e9))
}
```

```bash
> go test -race -run TestTimerRaceCondition
529.648825ms
==================
WARNING: DATA RACE
Read at 0x00c00011e090 by goroutine 9:
  github.com/quii/learn-go-with-tests/sync/v3_test.TestTimerRaceCondition.func1()
      /home/nathan/Project/learn-go-with-tests/sync/v3/timer_test.go:18 +0xd3

Previous write at 0x00c00011e090 by goroutine 7:
  github.com/quii/learn-go-with-tests/sync/v3_test.TestTimerRaceCondition()
      /home/nathan/Project/learn-go-with-tests/sync/v3/timer_test.go:16 +0x159
  testing.tRunner()
      /usr/local/go/src/testing/testing.go:1792 +0x225
  testing.(*T).Run.gowrap1()
      /usr/local/go/src/testing/testing.go:1851 +0x44

Goroutine 9 (running) created at:
  time.goFunc()
      /usr/local/go/src/time/sleep.go:215 +0x44

Goroutine 7 (running) created at:
  testing.(*T).Run()
      /usr/local/go/src/testing/testing.go:1851 +0x8f2
  testing.runTests.func1()
      /usr/local/go/src/testing/testing.go:2279 +0x85
  testing.tRunner()
      /usr/local/go/src/testing/testing.go:1792 +0x225
  testing.runTests()
      /usr/local/go/src/testing/testing.go:2277 +0x96c
  testing.(*M).Run()
      /usr/local/go/src/testing/testing.go:2142 +0xeea
  main.main()
      _testmain.go:51 +0x164
==================
```

也能對另外兩個執行測試，結果都是 Pass
```bash
go test -race -run TestTimerWithMutex
go test -race -run TestTimerWithRecursiveFunc
```

在錯誤報告中明確指出，在 `t.Reset(randomDuration())` 讀取了 `t`。且在main goroutine 中，`t = time.AfterFunc(...)` 修改了 `t`。

main goroutine：建立一個變數 t，然後建立一個timer，並"打算"把timer 賦值给t
Timer goroutine：timer可能會非常快地觸發，並開始執行callback function，callback function嘗試使用變數t

問題就在於這是一場"賽跑"：
- 如果main goroutine先完成赋值操作，那麼一切正常
- 如果Timer goroutine比較"積極快速"，可能在t被正確賦值之前就嘗試使用它，此時就會發生race condition。

這是一個典型的資料競爭場景：一個變數（t）被多個協程同時訪問，其中至少一個是寫入操作，而這些操作之間沒有同步機制。

## Go dead lock 分析

## More Go 併發練習