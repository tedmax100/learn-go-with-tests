# Mocking 模擬替身
---

在 DI 章節提到 DOC 我們除了模擬輸出結果外，有時也會需要檢測 SUT 與 DOC 交互情況來做為測試標準。為此 mock 通常會提供紀錄調用次數、throw exception、斷言傳入參數等的能力。

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

光測試交互次數，交互順序也不夠，還想要驗證睡眠時間。

# 自動產生 Mock

- [Gomock](https://github.com/uber-go/mock)

# Fuzz Test
https://tonybai.com/2021/12/01/first-class-fuzzing-in-go-1-18/