# Generics 泛型

---

# 簡單總結

###  如果用 any？
當我們用 interface{} 代表「任意型別」時，編譯器無法得知實際儲存的資料型別。
取出值後常需做「型別斷言」（type assertion），否則無法對其做任何運算：
那麼開發者肯定要
```go
func Pop() interface{} { … }

// 使用 Pop 的人必須要
v, ok := Pop().(int)
if !ok { /* 錯誤處理 */ }
sum := v + 1  // 僅在確定為 int 時才能使用 +
```

如果斷言錯誤，就會在執行時發生 panic，且漏寫斷言或錯誤判斷都可能造成隱藏錯誤。

###  改成用 Generics？
在宣告函式或資料結構時，加入「型別參數」T，並可加上約束（Constraint），如 comparable、any 等
```go
type Stack[T any] struct {
  values []T
}
```

呼叫時直接寫明或由編譯器推斷型別：
```go
si := NewStack[int]()    // Stack[int]
ss := NewStack[string]() // Stack[string]
```

所以就能提供編譯期檢查

- 同一性保障：Stack[int] 只接受 int，絕不會塞入 string
- 零斷言成本：Pop() 回傳的即是 T（例如 int），毋需再做型別斷言
- 運算安全：因為有約束（例如 comparable）, 可直接使用 ==、!=；若想用 +，可改用自訂約束

Example：禁止「蘋果＋橘子」
```go
si := NewStack[int]()
si.Push(1)
// si.Push("hello") // 編譯錯誤：cannot use "hello" (type string) as type int in argument

```

泛型將「型別」從執行時的動態檢查提前到編譯時，避免了大量的型別斷言和執行時錯誤。
在維持抽象和重用的同時，保證了程式碼的 強型別，使程式更安全、更易維護。

這重構的過程，絕對需要測試在保護的！

[GO Generic 入門筆記](https://ganhua.wang/go-generic)
