# Mocking 模擬替身
---

在 DI 章節提到 DOC 我們除了模擬輸出結果外，有時也會需要檢測 SUT 與 DOC 交互情況來做為測試標準。為此 mock 通常會提供紀錄調用次數、throw exception、斷言傳入參數等的能力。

範例v1與v2其實在 DI 章節，就演示過類似的東西了。

## 範例 v3
這次的測試對象是 `Countdown`除了想測試它輸出的結果，也想檢測調用 `sleeper.Sleep()` 的次數是否如預期。
```go
const finalWord = "Go!"
const countdownStart = 3

// Countdown prints a countdown from 3 to out with a delay between count provided by Sleeper.
func Countdown(out io.Writer, sleeper Sleeper) {
	for i := countdownStart; i > 0; i-- {
		fmt.Fprintln(out, i)
		sleeper.Sleep()
	}

	fmt.Fprint(out, finalWord)
}
```

實做 `SpySleeper`，並紀錄調用次數。並檢查 SpySleeper 被 Countdown 調用的次數是否3次。
```go
func TestCountdown(t *testing.T) {
	// Arrange：準備測試環境和測試數據
	buffer := &bytes.Buffer{}
	spySleeper := &SpySleeper{}

	// Act：執行被測試的程式碼
	Countdown(buffer, spySleeper)

	// Assert：驗證測試結果
	got := buffer.String()
	want := `3
2
1
Go!`

	if got != want {
		t.Errorf("got %q want %q", got, want)
	}

	if spySleeper.Calls != 3 {
		t.Errorf("not enough calls to sleeper, want 3 got %d", spySleeper.Calls)
	}
}

type SpySleeper struct {
	Calls int
}

func (s *SpySleeper) Sleep() {
	s.Calls++
}
```

## 範例 v4
**Still some problems**
除了測試次數，還想檢驗執行順序。Sleep 跟 Countdown 的執行順序希望如下。
因為sleep 剛剛測試了確實發生了三次。但它們的執行順序也可能不如預期。
```
Print N
Sleep
Print N-1
Sleep
Print Go!
```

不信？把 Countdown 改成以下，剛剛的測試還是會過得，但這顯然不是我們要的。
```go
func Countdown(out io.Writer, sleeper Sleeper) {
	for i := countdownStart; i > 0; i-- {
		sleeper.Sleep()
	}

	for i := countdownStart; i > 0; i-- {
		fmt.Fprintln(out, i)
	}

	fmt.Fprint(out, finalWord)
}
```

修改 `SpyCountdownOperations`，同時實作了 io.Writer 和 Sleeper ，將每個呼叫記錄為一個slice。
```go
type SpyCountdownOperations struct {
	Calls []string
}

func (s *SpyCountdownOperations) Sleep() {
	s.Calls = append(s.Calls, sleep)
}

func (s *SpyCountdownOperations) Write(p []byte) (n int, err error) {
	s.Calls = append(s.Calls, write)
	return
}

const write = "write"
const sleep = "sleep"
```

測試程式在 printf 跟 sleep 都傳入 spySleepPrinter。不同介面的實做會呼叫不同的func，來做執行過程的紀錄．
```go
t.Run("sleep before every print", func(t *testing.T) {
		spySleepPrinter := &SpyCountdownOperations{}
		Countdown(spySleepPrinter, spySleepPrinter)

		want := []string{
			write,
			sleep,
			write,
			sleep,
			write,
			sleep,
			write,
		}

		if !reflect.DeepEqual(want, spySleepPrinter.Calls) {
			t.Errorf("wanted calls %v got %v", want, spySleepPrinter.Calls)
		}
	})
```

## 範例 v5
光測試交互次數，交互順序也不夠，還想要驗證睡眠時間。
建立一個 `ConfigurableSleeper` 類別，來保管睡眠時間設定。還有一個 `sleep func`，入參是 duration，這樣的簽章也是 Go time 標準庫中的 sleep 簽章。
```go
package time

// Sleep pauses the current goroutine for at least the duration d.
// A negative or zero duration causes Sleep to return immediately.
func Sleep(d Duration)
```

```go
type ConfigurableSleeper struct {
	duration time.Duration
	sleep    func(time.Duration)
}

// Sleep will pause execution for the defined Duration.
func (c *ConfigurableSleeper) Sleep() {
	c.sleep(c.duration)
}
```

為了測試建立一個 SpyTime 類別，到時就用這 SpyTime 實例，來注入到 ConfigurableSleeper 中。
這裡用的是**建構式注入**策略。
```go
type SpyTime struct {
	durationSlept time.Duration
}

func (s *SpyTime) Sleep(duration time.Duration) {
	s.durationSlept = duration
}

func TestConfigurableSleeper(t *testing.T) {
	sleepTime := 5 * time.Second

	spyTime := &SpyTime{}
	sleeper := ConfigurableSleeper{sleepTime, spyTime.Sleep}
	sleeper.Sleep()

	if spyTime.durationSlept != sleepTime {
		t.Errorf("should have slept for %v but slept for %v", sleepTime, spyTime.durationSlept)
	}
}
```

而一般客戶端使用時就如下，直接注入標準庫的 `time.Sleep` 即可。
```go
func main() {
	sleeper := &ConfigurableSleeper{1 * time.Second, time.Sleep}
	Countdown(os.Stdout, sleeper)
}
```

這樣就能透過 ConfigurableSleeper 來自由設定睡眠時間以及 sleeper 實例，來進行測試。



# 自動產生 Mock

- [Gomock](https://github.com/uber-go/mock)

# 總結
Mocking 本身是蠻奇妙的東西，它完全是模擬出來的。甚至我們很可能為了讓自己的設計能夠被測試需要去在一個測試案例中 mock 很多依賴物件，但這行為是個值得思考的訊號。
- 可能你的測試對象的設計，職責不夠單一，導致需要依賴很多物件，所以你測試才需要很多mock的動作。（文章說超過3個需要mock，代表能重新思考設計了）
- 或者你依賴物件的顆粒過於細緻，其實可以整併成一個 interface 來注入使用，還方便 mock 框架來產生mock物件。
- 測試時的著力點施力錯誤，我們應專住在測試對象（SUT）的行為上，而不是想進辦法測試依賴物件（DOC）的實做細節。例如資料庫怎麼儲存、怎麼搞定transaction那些的，其實在單元測試中並不需要去實做它。
- 可能我們對於這模組/功能在設計其抽象時，就設計錯誤了。
- 別想著要去測試 private func，因為它是細節，你應該去測試能被測試案例調用的 public func。

# Fuzz Test
https://tonybai.com/2021/12/01/first-class-fuzzing-in-go-1-18/


> 範例 `v6`，「**Bonus go 1.23 Iterator**」 留至 Generic 章節在分享。
> 這需要泛型的概念。