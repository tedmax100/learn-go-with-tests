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

> waitgroup 本身由 `counter`、`waiters`和 `semaphore`組成。Add 可以給正數或負數，這裡會直接對 counter 加減，如果 counter 為 0 表示goroutine 所有任務已經完成，就會去喚醒所有被阻塞在這等待的 goroutine （waiter）起床。`semaphore`目的是用來阻塞跟喚醒goroutine用的。
> 所以在這例子中，`wg.Add(wantedCount)` 先替 counter 加了 1000，`wg.Done()`時就開始替counter -1。而 測試函數本身這goroutine 則被迫阻塞在等待 `wg.Wait()`來喚醒它。

執行看看測試會發現第二個併發的測試出錯了，不是想像中的 `1000`。

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
  - 如果兩個 goroutine 同時 `讀取到相同的值`，然後都加 1 再寫回，結果只增加了 `1` 而不是 `2`。

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

Go 的 sync 標準庫提了了很多基礎類型的原子操作。這裡我們能使用 `atomic.AddInt64()`與 `atomic.LoadInt64()`來處理。

> 原子操作（Atomic Operation）就是那3步驟是不可被別的gorotine給執行過程中不會被中斷或干擾。
> 也就是3個操作要麼全部完成，要麼完全不做（不會停在中間狀態）。也不會和其他原子操作互相干擾，避免race condition。
> atomic 保證這個操作在多核、多執行緒環境中看起來就像是「單執行緒」在執行這個加法——也就是說，這個操作是「不可中斷」且「線性化」的（linearizable）。
> 換句話說，雖然實際上有很多 goroutine 同時執行，但在操作這個特定變數時，CPU 會確保：
>
> 1. 每次只有一個 goroutine 的 AddInt64 操作能成功執行。
> 2. 其他 goroutine 必須排隊等待前一個操作完成，才能執行下一個。
>    這種排隊和互斥是由 CPU 的原子指令和快取一致性協議（cache coherence protocol）硬體層面保證的。

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

加入一個新的測試對象是 `AtomicCounter`，但跑一樣的測試方法。

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

Go的死鎖情況可能非常隱蔽，特別是在使用多個 lock 和 channel 時。以下我們將探討一些更複雜的死鎖場景。

### 讀寫鎖的隱形死鎖

讀寫鎖(sync.RWMutex)是Go中常用的同步原語，但如果使用不當，容易形成死鎖。以下是一個經典的隱形死鎖示例：

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var rwMutex sync.RWMutex
	ch := make(chan int) // unbuffered channel

	// goroutine 1：hold write lock，send msg to channel
	go func() {
		rwMutex.Lock() 
		fmt.Println("goroutine1：獲取寫鎖，準備發送到通道")
		ch <- 123  // 阻塞等待接收者
		fmt.Println("goroutine1：發送完成，釋放寫鎖")
		rwMutex.Unlock()
	}()

	time.Sleep(100 * time.Millisecond) // 確保goroutine1先執行

	// goroutine2：try to octain read lock, and receive msg from channel
	go func() {
		fmt.Println("goroutine2：嘗試獲取讀鎖")
		rwMutex.RLock() // 阻塞，因為協程1持有寫鎖
		fmt.Println("goroutine2：獲取到讀鎖，準備從通道接收")
		x := <-ch
		fmt.Println("goroutine2：讀到", x)
		rwMutex.RUnlock()
	}()

	time.Sleep(2 * time.Second)
	fmt.Println("結束")
}
```

這是一個典型的死鎖情況：

1. 協程1獲取寫鎖，然後嘗試發送到通道，但因為無人接收而阻塞
2. 協程2嘗試獲取讀鎖，但因為協程1持有寫鎖而阻塞
3. 結果：協程1等待協程2接收數據，而協程2等待協程1釋放鎖，形成循環等待

### 多鎖死鎖（鎖順序問題）

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var lockA, lockB sync.Mutex

	// goroutine 1：先獲取A鎖，然後嘗試獲取B鎖
	go func() {
		lockA.Lock()
		fmt.Println("goroutine 1：獲取鎖A")
		time.Sleep(100 * time.Millisecond) // 模擬工作
		fmt.Println("goroutine 1：嘗試獲取鎖B")
		lockB.Lock()
		fmt.Println("goroutine 1：獲取到鎖B")
		// 操作共享資源
		lockB.Unlock()
		lockA.Unlock()
	}()

	// goroutine 2：先獲取B鎖，然後嘗試獲取A鎖（順序與協程1相反）
	go func() {
		lockB.Lock()
		fmt.Println("goroutine 2：獲取鎖B")
		time.Sleep(100 * time.Millisecond) // 模擬工作
		fmt.Println("goroutine 2：嘗試獲取鎖A")
		lockA.Lock()
		fmt.Println("goroutine 2：獲取到鎖A")
		// 操作共享資源
		lockA.Unlock()
		lockB.Unlock()
	}()

	time.Sleep(2 * time.Second)
	fmt.Println("可能已死鎖")
}
```

這種情況是經典的"資源獲取順序"問題：

1. 協程1獲取鎖A，然後嘗試獲取鎖B
2. 同時協程2獲取鎖B，然後嘗試獲取鎖A
3. 兩個協程互相等待對方釋放鎖，形成死鎖

### 混合使用多種同步原語

更複雜的死鎖可能涉及多種同步原語的混合使用：

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var mutex sync.Mutex
	cond := sync.NewCond(&mutex)
	ready := false

	// 控制協程
	go func() {
		time.Sleep(100 * time.Millisecond)
		mutex.Lock()
		// 在沒有調用Broadcast的情況下解鎖
		// 忘記了喚醒等待的協程
		ready = true
		mutex.Unlock()
		fmt.Println("控制協程：狀態已設置，但忘記發送信號")
	}()

	// 工作協程
	for i := 0; i < 3; i++ {
		go func(id int) {
			mutex.Lock()
			for !ready {
				fmt.Printf("工作協程%d：等待信號\n", id)
				cond.Wait() // 等待條件變量信號
			}
			fmt.Printf("工作協程%d：收到信號，開始工作\n", id)
			mutex.Unlock()
		}(i)
	}

	time.Sleep(3 * time.Second)
	fmt.Println("程序結束：工作協程可能永遠被阻塞")
}
```

在這個例子中：

1. 工作協程等待條件變量信號
2. 控制協程設置了條件，但忘記調用 `cond.Broadcast()`或 `cond.Signal()`
3. 工作協程將永遠阻塞在 `cond.Wait()`

避免死鎖的最佳實踐：

- 保持一致的鎖獲取順序
- 避免在持有鎖時進行阻塞操作
- 優先使用通道進行同步而非鎖
- 使用帶timeout 的 context 或 select default 語句避免永久阻塞
- 定期審查並發程式邏輯

Channel 亂用也是有可能 race condition/deadlock

> When to use locks over channels and goroutines?
>
> - Use channels when passing ownership of data
> - Use mutexes for managing state，也因為 mudex 是用來實現互斥鎖的，所以別 copy 它。
>   如果將包含 sync.Mutex 的對象pass by value（這等於複製了一份mutex），也會複製互斥鎖的內部狀態，這會導致︰
> - 鎖的狀態被複製，兩個不同的鎖實例實際上指向同一個底層資源，很可能導致鎖的行為異常
> - 也可能因為被複製，出現dead lock、data racing 等難以除錯的問題
> - `go vet` 工具可以檢測這種複製的問題，並提供錯誤提醒。

```go
package v5

import (
	"sync"
)

type Counter struct {
	mu    sync.Mutex
	value int
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}
```

```bash
go vet ./...
# github.com/quii/learn-go-with-tests/sync/v5
# [github.com/quii/learn-go-with-tests/sync/v5]
./counter_test.go:9:38: assertCounter passes lock by value: github.com/quii/learn-go-with-tests/sync/v5.Counter contains sync.Mutex
./counter_test.go:24:20: call of assertCounter copies lock value: github.com/quii/learn-go-with-tests/sync/v5.Counter contains sync.Mutex
./counter_test.go:42:20: call of assertCounter copies lock value: github.com/quii/learn-go-with-tests/sync/v5.Counter contains sync.Mutex
(base) 
```

# Bonus SyncTest

SyncTest 使用虛擬時鐘和gorotuine group(又稱 bubble) 來控制併發程式碼的執行時序，使得測試記快速又可靠。

Run 方法在一個新的goroutine 中執行 func f()，並建立一個獨立的bubble，確保所有相關的gorotuine都在虛擬時鐘的控制下執行。

Wait 用來同步 bubble 中 goroutine 的狀態。調用wait後，main goroutine 將被block，直到 bubble 中的其他goroutine 狀態都處於 durably  blocked 狀態。

以下測試場景是
1. 在 cancel 之前, 測試目標 function 沒被調用
2. 測試目標 function是在 cancel 之後才被呼叫
```go
func TestAfterFunc(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())

    calledCh := make(chan struct{}) // closed when AfterFunc is called
    context.AfterFunc(ctx, func() {
        close(calledCh)
    })

    // TODO: Assert that the AfterFunc has not been called.

    cancel()

    // TODO: Assert that the AfterFunc has been called.
}
```
這樣的測試需求下，通常只能等待一段時間，再來判斷事件會不會發生。以及使用 helper function 來紀錄該function 的調用情況。這樣的測試非常的慢，在這類型的測試越來越多時，整個單元測試的執行時間會越來越久。完整程式能看 `v6` 資料夾。
```go
// funcCalled reports whether the function was called.
funcCalled := func() bool {
    select {
    case <-calledCh:
        return true
    case <-time.After(10 * time.Millisecond):
        return false
    }
}

if funcCalled() {
    t.Fatalf("AfterFunc function called before context is canceled")
}

cancel()

if !funcCalled() {
    t.Fatalf("AfterFunc function not called after context is canceled")
}
```

透過 testing/synctest 重構測試程式
```go
func TestAfterFunc(t *testing.T) {
	// 將整個測試程式包在 Run 之中，建立一個獨立的bubble以及使用虛擬時鐘。
    synctest.Run(func() {
        ctx, cancel := context.WithCancel(context.Background())

        funcCalled := false
        context.AfterFunc(ctx, func() {
            funcCalled = true
        })

		// Wait 會等待 bubble 的每個 goroutine 都 blocked。
		// 一個 goroutine 處於阻塞狀態時，意味著它當前不執行任何代碼，而是在等待某些事件發生，比如：	
		// 等待 channel 操作完成
		// 等待 lock 釋放
		// 等待 timer 觸發
		// 等待 I/O 操作等
        synctest.Wait()
		// 當 synctest.Wait() 返回時，我們可以確定系統已達到穩定狀態，意味著以下兩種情況之一：
		// context 已經調用了註冊的 callback function
		// context 還沒有調用 callback function，且在我們執行下一步操作（如 cancel）前不會調用
		// 這個機制讓我們能夠在測試中確定地檢查系統狀態，而不需要使用 time.Sleep() 這種不可靠的方式等待異步操作。這大大提高了測試的可靠性和速度，因為我們不需要猜測要等待多長時間才能確保某個操作完成或未完成。
        if funcCalled {
            t.Fatalf("AfterFunc function called before context is canceled")
        }

        cancel()

        synctest.Wait()
        if !funcCalled {
            t.Fatalf("AfterFunc function not called after context is canceled")
        }
    })
}
```

測試案例 2: 測試 Time
```go
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
```

使用 synctest 重構
```go
func TestWithTimeout(t *testing.T) {
    synctest.Run(func() {
        const timeout = 5 * time.Second
        ctx, cancel := context.WithTimeout(context.Background(), timeout)
        defer cancel()

        // Wait just less than the timeout.
        time.Sleep(timeout - time.Nanosecond)

		// 我們撰寫這個測試，就像使用真實時鐘一樣。
		// 唯一不同的是，我們將測試包裝在 synctest.Run 中，並在每次呼叫 time.Sleep 後呼叫 synctest.Wait，以等待 context 的 timer 執行完畢。
        synctest.Wait()
        if err := ctx.Err(); err != nil {
            t.Fatalf("before timeout, ctx.Err() = %v; want nil", err)
        }

        // Wait the rest of the way until the timeout.
        time.Sleep(time.Nanosecond)
        synctest.Wait()
        if err := ctx.Err(); err != context.DeadlineExceeded {
            t.Fatalf("after timeout, ctx.Err() = %v; want DeadlineExceeded", err)
        }
    })
}

這用了  g 
```



https://huizhou92.com/zh-cn/p/go-1.24-%E6%96%B0%E7%9A%84%E5%AE%98%E6%96%B9%E5%BA%93-synctest/

https://zhuanlan.zhihu.com/p/405699766
https://zhuanlan.zhihu.com/p/375530785
https://brantou.github.io/2017/05/23/go-race-detector/
