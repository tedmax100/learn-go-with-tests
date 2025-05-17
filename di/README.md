# Dependency Injection 依賴注入

---

# Before

## SUT 測試對象 （System Under Test）
```go
func Greet(name string) {
	fmt.Printf("Hello, %s", name)
}
```

`怎麼測試？`
`測試標準是什麼？`

- 假如要測試標準是否輸出 `Hello, ${name}`
- 或是想驗收是否調用輸出對象一次

![](./sut_doc.png)

`func Greet(name string)` 怎測試？
1. 該方法沒回傳值（no return），直接會輸出至`stdout`，沒法在程式中獲取來檢測
2. 該方法內容直接依賴 `fmt.Printf` （各系統的終端輸出）
   1. 屬於耦合等級中的 `external coupling`，屬於不可控制的部份
   2. 而 `fmt.Printf` 屬於這測試對象的 **DOC (Depended On Component) **

為了能提高這測試對象的可測試性，使其能夠被測試。
能使用 `依賴注入`

## 思路

### 依賴注入
`依賴注入`是一種設計方法，可以讓測試對象更加靈活且可測試。其主要原則是：
將測試對象能用以下方式來進行替換
- 建構式時注入 Constructor Injection
- 屬性注入 (Property Injection)
- 方法入參注入 Method Injection

> 操作: 追蹤`fmt.Printf` 的實做至 Fprintf

```go
// It returns the number of bytes written and any write error encountered.
func Printf(format string, a ...interface{}) (n int, err error) {
	return Fprintf(os.Stdout, format, a...)
}

func Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	p := newPrinter()
	p.doPrintf(format, a)
	n, err = w.Write(p.buf)
	p.free()
	return
}

type Writer interface {
	Write(p []byte) (n int, err error)
}
```

> Our function doesn't need to care *where* or *how* the printing happens, so we should accept an *interface* rather than a concrete type.
> 但其實這方法並不在意這輸出至哪裡甚至如何輸出，為此能考慮傳入一個抽象或介面給該對象使用。這樣就能輸出到我們`可以控制`的元件中進行檢測。
> `Fprintf` 能發現它也是使用**方法入參注入**，將需要輸出的傳入一組滿足此介面的物件裡，且此時該物件是調用方可以控制其狀態與能力。所以能考慮把 `Fprintf` 上提，來取代原本使用的 `fmt.Printf`。

因為測試對象是單純只有個 method，所以重構策略採用"方法入參注入"。戰術設計上直接藉由 Go 標準IO庫提供的 Writer 介面來作為入參對象。

# After 
將`Greeter`  重構變成入參使用 Writer 介面做注入。fmt.Printf 改成用 fmt.Fprintf，來將字串給輸出至指定的writer。
```go
func Greet(writer io.Writer, name string) {
	fmt.Fprintf(writer, "Hello, %s", name)
}
```
那我們就能開始寫測試了。此時測試案例設定是想要驗收輸出的字串長相是否如預期。

```
func TestGreet(t *testing.T) {
	// Arrange：準備測試環境和測試數據
	buffer := bytes.Buffer{}

	// Actt：執行 SUT
	Greet(&buffer, "Chris")
	got := buffer.String()
	
	// Assert：驗證結果是否符合預期
	want := "Hello, Chris"

	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
```

```bash
> go test -timeout 30s -run ^TestGreet$ github.com/quii/learn-go-with-tests/di/v1

ok  	github.com/quii/learn-go-with-tests/di/v1	0.002s
```

```mermaid
classDiagram
    class ioWriter {
        <<interface>>
        +Write(p []byte) (int, error)
    }
    
    class bytesBuffer {
        +String() string
        +Write(p []byte) (int, error)
    }
    
    class Greet {
        <<function>>
        +Greet(writer io.Writer, name string) void
    }
    
    class TestGreet {
        <<function>>
        +TestGreet(t *testing.T) void
    }
    
    bytesBuffer ..|> ioWriter : implements
    Greet --> ioWriter : depends on
    TestGreet --> bytesBuffer : creates & uses
    TestGreet --> Greet : calls with buffer
```

## 重構客戶端

### Before
此時這裡會無法編譯，因為方法簽章的調用不滿足已經重構過得 Greet
```go
func main() {
	Greet("Elodie")
}
```

### After
透過方法入參注入，客戶端可以選擇注入對象，這裡選擇 `os.Stdout`。
```go
func main() {
	Greet(os.Stdout, "Elodie")
}
```

# Think More
要是沒有使用注入的設計策略， Greet 未來想要輸出至各種不同對象時，你可能會怎做？

1. Use Java or C#︰可能會用 method overloading？
```java
// Java 範例：使用方法重載處理不同輸出對象
public class Greeter {
    // 輸出到標準輸出
    public void greet(String name) {
        System.out.printf("Hello, %s", name);
    }
    
    // 輸出到文件
    public void greet(String name, File file) throws IOException {
        try (FileWriter writer = new FileWriter(file)) {
            writer.write(String.format("Hello, %s", name));
        }
    }
    
    // 輸出到StringBuilder
    public void greet(String name, StringBuilder builder) {
        builder.append(String.format("Hello, %s", name));
    }
    
    // 輸出到自定義輸出流
    public void greet(String name, PrintStream stream) {
        stream.printf("Hello, %s", name);
    }
}

// 客戶端使用
public class Client {
    public static void main(String[] args) throws IOException {
        Greeter greeter = new Greeter();
        
        // 輸出到 stdout
        greeter.greet("Elodie");
        
        // 輸出到文件
        greeter.greet("Elodie", new File("greeting.txt"));
        
        // 輸出到 StringBuilder
        StringBuilder sb = new StringBuilder();
        greeter.greet("Elodie", sb);
        String result = sb.toString();
    }
}
```
2. Use Go︰可能會新增一堆不同名稱的 method？
```go
// Go 範例：使用不同名稱的函數處理不同輸出對象
package greet

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// 輸出到標準輸出
func Greet(name string) {
	fmt.Printf("Hello, %s", name)
}

// 輸出到文件
func GreetToFile(name string, filename string) error {
	content := fmt.Sprintf("Hello, %s", name)
	return ioutil.WriteFile(filename, []byte(content), 0644)
}

// 輸出到字符串
func GreetToString(name string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Hello, %s", name)
	return sb.String()
}

// 輸出到錯誤輸出
func GreetToStderr(name string) {
	fmt.Fprintf(os.Stderr, "Hello, %s", name)
}

// 客戶端使用
func main() {
	// 輸出到標準輸出
	Greet("Elodie")
	
	// 輸出到文件
	_ = GreetToFile("Elodie", "greeting.txt")
	
	// 獲取字符串形式
	message := GreetToString("Elodie")
	
	// 輸出到錯誤輸出
	GreetToStderr("Elodie")
}
```

這兩種方式相比使用依賴注入的方式，都存在以下問題：

1. 程式碼重複：每個方法都包含相同的問候邏輯，違反了 DRY 原則
2. 測試困難：每個方法可能需要單獨的測試用例
3. 可擴展性差：新增輸出目標時需要修改原始程式碼
4. 職責混亂：問候邏輯與輸出機制耦合在一起

使用依賴注入後，程式碼更簡潔（這裡的簡潔指的是認知負載==可閱讀性+可維護性）、更具可測試性和擴展性，因為它將"做什麼"（問候邏輯）與"如何做"（輸出方式）分離開來。

## I/O
實務中的 I/O 有很多種，寫檔案、輸出至stdout、網路網路

因為 Go http 標準庫中提供了 ResponseWriter 介面，該介面也剛好定義了io.Writer介面所需。

```go
// A ResponseWriter may not be used after [Handler.ServeHTTP] has returned.
type ResponseWriter interface {
	Header() Header

	Write([]byte) (int, error)

	WriteHeader(statusCode int)
}
```

因為我們已經將 Greet 重構成使用 io.Writer 介面。所以能很快擴展成輸出至 WebAPI 的輸出格式與處理物件 `http.ResponseWriter`。
```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func Greet(writer io.Writer, name string) {
	fmt.Fprintf(writer, "Hello, %s", name)
}

func MyGreeterHandler(w http.ResponseWriter, r *http.Request) {
	Greet(w, "world")
}

func main() {
	log.Fatal(http.ListenAndServe(":5001", http.HandlerFunc(MyGreeterHandler)))
}
```

軟體系統中經常依賴但不可控制的還有以下類型︰
## Time & Date
時間是軟體開發中另一個常見的外部依賴，也經常會造成測試困難。以下方式可以通過依賴注入改善時間相關功能的可測試性：

**問題**
考慮一個需要使用當前時間的函數：
```go
// 直接依賴系統時間
func GreetWithTime(name string) string {
    currentTime := time.Now()
    if currentTime.Hour() < 12 {
        return fmt.Sprintf("早安, %s", name)
    } else if currentTime.Hour() < 18 {
        return fmt.Sprintf("午安, %s", name)
    }
    return fmt.Sprintf("晚安, %s", name)
}
```

這種實現方式有幾個問題：
- 無法在測試中控制時間，導致測試結果可能會根據執行時間變化
- 無法測試所有的時間場景（早上、下午、晚上）
- 時間相關的邏輯無法被隔離測試

**解決方案**
1. 方法入參注入
```go
// 使用方法入參注入時間
func GreetWithTime(name string, currentTime time.Time) string {
    if currentTime.Hour() < 12 {
        return fmt.Sprintf("早安, %s", name)
    } else if currentTime.Hour() < 18 {
        return fmt.Sprintf("午安, %s", name)
    }
    return fmt.Sprintf("晚安, %s", name)
}
```
測試：
```go
func TestGreetWithTime(t *testing.T) {
    // 測試早上問候
    t.Run("morning greeting", func(t *testing.T) {
        morningTime := time.Date(2023, time.January, 1, 8, 0, 0, 0, time.UTC)
        result := GreetWithTime("Chris", morningTime)
        expected := "早安, Chris"
        if result != expected {
            t.Errorf("Expected %q but got %q", expected, result)
        }
    })
    
    // 測試下午問候
    t.Run("afternoon greeting", func(t *testing.T) {
        afternoonTime := time.Date(2023, time.January, 1, 15, 0, 0, 0, time.UTC)
        result := GreetWithTime("Chris", afternoonTime)
        expected := "午安, Chris"
        if result != expected {
            t.Errorf("Expected %q but got %q", expected, result)
        }
    })
    
    // 測試晚上問候
    t.Run("evening greeting", func(t *testing.T) {
        eveningTime := time.Date(2023, time.January, 1, 20, 0, 0, 0, time.UTC)
        result := GreetWithTime("Chris", eveningTime)
        expected := "晚安, Chris"
        if result != expected {
            t.Errorf("Expected %q but got %q", expected, result)
        }
    })
}
```

2. 依賴抽象interface

對於更複雜的場景，可以定義一個時間提供者接口：
```go
// TimeProvider 提供獲取時間的介面
type TimeProvider interface {
    Now() time.Time
}

// RealTimeProvider 實現真實的時間獲取
type RealTimeProvider struct{}

func (p *RealTimeProvider) Now() time.Time {
    return time.Now()
}

// 測試用的固定時間提供者
type StubTimeProvider struct {
    FixedTime time.Time
}

func (p *StubTimeProvider) Now() time.Time {
    return p.FixedTime
}

// 使用時間提供者的問候函數
func GreetWithTimeProvider(name string, timeProvider TimeProvider) string {
    currentTime := timeProvider.Now()
    if currentTime.Hour() < 12 {
        return fmt.Sprintf("早安, %s", name)
    } else if currentTime.Hour() < 18 {
        return fmt.Sprintf("午安, %s", name)
    }
    return fmt.Sprintf("晚安, %s", name)
}
```

在測試中：
```go
func TestGreetWithTimeProvider(t *testing.T) {
    // 測試早上問候
    t.Run("morning greeting", func(t *testing.T) {
        morningTime := time.Date(2023, time.January, 1, 8, 0, 0, 0, time.UTC)
        stubTimeProvider := &StubTimeProvider{FixedTime: morningTime}
        result := GreetWithTimeProvider("Chris", stubTimeProvider)
        expected := "早安, Chris"
        if result != expected {
            t.Errorf("Expected %q but got %q", expected, result)
        }
    })
    
    // 測試下午問候
    t.Run("afternoon greeting", func(t *testing.T) {
        afternoonTime := time.Date(2023, time.January, 1, 15, 0, 0, 0, time.UTC)
        stubTimeProvider := &StubTimeProvider{FixedTime: afternoonTime}
        result := GreetWithTimeProvider("Chris", stubTimeProvider)
        expected := "午安, Chris"
        if result != expected {
            t.Errorf("Expected %q but got %q", expected, result)
        }
    })
}
```

在實際應用中：
```go
func main() {
    realTimeProvider := &RealTimeProvider{}
    greeting := GreetWithTimeProvider("Elodie", realTimeProvider)
    fmt.Println(greeting)
}
```

```mermaid
classDiagram
    class TimeProvider {
        <<interface>>
        +Now() time.Time
    }
    
    class RealTimeProvider {
        +Now() time.Time
    }
    
    class StubTimeProvider {
        +FixedTime time.Time
        +Now() time.Time
    }
    
    class GreetWithTimeProvider {
        <<function>>
        +GreetWithTimeProvider(name string, timeProvider TimeProvider) string
    }
    
    class TestGreetWithTimeProvider {
        <<function>>
        +TestGreetWithTimeProvider(t *testing.T)
    }
    
    class Main {
        <<function>>
    }
    
    RealTimeProvider ..|> TimeProvider : implements
    StubTimeProvider ..|> TimeProvider : implements
    GreetWithTimeProvider --> TimeProvider : depends on
    TestGreetWithTimeProvider --> StubTimeProvider : creates
    TestGreetWithTimeProvider --> GreetWithTimeProvider : calls with stub
    Main --> RealTimeProvider : creates
    Main --> GreetWithTimeProvider : calls with real
```

**時間相關測試的最佳實踐**
1. 永遠不要直接使用 `time.Now()`：哪怕需求是**現在**，如果在函數內部直接調用，將會無法測試，可以通過依賴注入傳入
2. 定義 TimeProvier 介面：使代碼更容易進行單元測試
3. 考慮時區問題：在測試中明確指定時區，避免測試在不同環境下結果不一致
4. 使用固定的時間點進行測試：確保測試結果的可重複性
5. 測試**邊界條件**：例如時間段切換的臨界點（11:59 vs 12:00）

通過這種方式，我們將時間依賴也轉變為可控的依賴，大大提高了系統的可測試性。

## Database 或其他外部服務
資料庫和外部服務是應用程式中最常見的外部依賴。直接在代碼中依賴這些外部系統會導致：

1. 測試變得緩慢：需要實際連接資料庫或外部API
2. 測試不可靠：外部系統可能暫時不可用
3. 環境依賴：需要預先設置特定環境
4. 測試難以並行：可能存在資源競爭
5. 難以測試邊界情況：特定錯誤或異常情況難以製造

### 資料庫依賴注入示例
####  Before（直接依賴資料庫）：
```go
package user

import (
    "database/sql"
    _ "github.com/lib/pq"
)

type UserService struct {
    db *sql.DB
}

// 建立連接
func NewUserService() (*UserService, error) {
    db, err := sql.Open("postgres", "postgres://user:password@localhost/mydb?sslmode=disable")
    if err != nil {
        return nil, err
    }
    return &UserService{db: db}, nil
}

// 獲取用戶
func (s *UserService) GetUser(id int) (string, error) {
    var name string
    err := s.db.QueryRow("SELECT name FROM users WHERE id = $1", id).Scan(&name)
    if err != nil {
        return "", err
    }
    return name, nil
}
```

`GetUser` 因為直接依賴資料庫做查詢，進而難以測試。通常就會需要local建立一組 database 來測試。 

#### After
**使用依賴注入重構：**
1. 首先，定義Repository介面：
```go
package user

// UserRepository 定義了訪問用戶數據的接口
type UserRepository interface {
    GetUser(id int) (string, error)
}
```

2. 實做Repository（真正的資料庫實現）：
```go
package user

import "database/sql"

// PostgresUserRepository 是UserRepository的PostgreSQL實現
type PostgresUserRepository struct {
    db *sql.DB
}

// NewPostgresUserRepository 建立PostgreSQL版本的倉庫
func NewPostgresUserRepository(connectionString string) (*PostgresUserRepository, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, err
    }
    
    // 測試連接
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return &PostgresUserRepository{db: db}, nil
}

// GetUser 從資料庫中獲取用戶
func (r *PostgresUserRepository) GetUser(id int) (string, error) {
    var name string
    err := r.db.QueryRow("SELECT name FROM users WHERE id = $1", id).Scan(&name)
    if err != nil {
        return "", err
    }
    return name, nil
}
```

3. 重構Service層，使用建構式注入Repository：

```go
package user

// UserService 提供用戶相關的業務邏輯
type UserService struct {
    repo UserRepository
}

// NewUserService 通過依賴注入建立服務
func NewUserService(repo UserRepository) *UserService {
    return &UserService{
        repo: repo,
    }
}

// GetUserGreeting 獲取用戶問候資訊
func (s *UserService) GetUserGreeting(id int) (string, error) {
    name, err := s.repo.GetUser(id)
    if err != nil {
        return "", err
    }
    
    return "Hello, " + name, nil
}
```

4. 測試程式，使用Mock對象：
在這 SUT 下，DOC是Repository，所以要做一個假的 Repository
```go
package user

import (
    "errors"
    "testing"
)

// MockUserRepository 是用於測試的模擬實現
type MockUserRepository struct {
    users map[int]string
    callCount map[string]int
    shouldError bool
}

// NewMockUserRepository 建立模擬倉庫
func NewMockUserRepository() *MockUserRepository {
    return &MockUserRepository{
        users: make(map[int]string),
        callCount: make(map[string]int),
        shouldError: false,
    }
}

// GetUser 模擬從資料庫獲取用戶
func (r *MockUserRepository) GetUser(id int) (string, error) {
    r.callCount["GetUser"]++
    
    if r.shouldError {
        return "", errors.New("模擬資料庫錯誤")
    }
    
    name, exists := r.users[id]
    if !exists {
        return "", errors.New("用戶不存在")
    }
    
    return name, nil
}

// AddUser 添加測試用戶
func (r *MockUserRepository) AddUser(id int, name string) {
    r.users[id] = name
}

// SetError 設置模擬錯誤
func (r *MockUserRepository) SetError(shouldError bool) {
    r.shouldError = shouldError
}

// GetCallCount 獲取方法調用次數
func (r *MockUserRepository) GetCallCount(methodName string) int {
    return r.callCount[methodName]
}

// 測試獲取問候語
func TestUserService_GetUserGreeting(t *testing.T) {
    // Arrange
    mockRepo := NewMockUserRepository()
    mockRepo.AddUser(1, "Chris")
    
    service := NewUserService(mockRepo)
    
    // Act
    greeting, err := service.GetUserGreeting(1)
    
    // Assert
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
    
    expectedGreeting := "Hello, Chris"
    if greeting != expectedGreeting {
        t.Errorf("Expected greeting %q, got %q", expectedGreeting, greeting)
    }
    
    // 驗證Repository被調用了一次
    if callCount := mockRepo.GetCallCount("GetUser"); callCount != 1 {
        t.Errorf("Expected GetUser to be called once, got %d", callCount)
    }
}

// 測試用戶不存在的情況
func TestUserService_GetUserGreeting_UserNotFound(t *testing.T) {
    // Arrange
    mockRepo := NewMockUserRepository()
    // 不添加任何用戶
    
    service := NewUserService(mockRepo)
    
    // Act
    _, err := service.GetUserGreeting(1)
    
    // Assert
    if err == nil {
        t.Error("Expected an error for non-existent user, got none")
    }
}

// 測試數據庫錯誤情況
func TestUserService_GetUserGreeting_DatabaseError(t *testing.T) {
    // Arrange
    mockRepo := NewMockUserRepository()
    mockRepo.AddUser(1, "Chris")
    mockRepo.SetError(true)
    
    service := NewUserService(mockRepo)
    
    // Act
    _, err := service.GetUserGreeting(1)
    
    // Assert
    if err == nil {
        t.Error("Expected a database error, got none")
    }
}
```

5. 在實際應用中的使用方式：
```go
package main

import (
    "fmt"
    "log"
    
    "myapp/user"
)

func main() {
    // 建立真實的資料庫repository
    repo, err := user.NewPostgresUserRepository("postgres://user:password@localhost/mydb?sslmode=disable")
    if err != nil {
        log.Fatalf("無法連接資料庫: %v", err)
    }
    
    // 注入repository到service
    service := user.NewUserService(repo)
    
    // 使用service
    greeting, err := service.GetUserGreeting(1)
    if err != nil {
        log.Fatalf("獲取用戶信息失敗: %v", err)
    }
    
    fmt.Println(greeting)
}
```

# 依賴注入處理外部依賴的最佳實踐
1. 使用 Interface 隔離外部依賴：定義清晰的interface，代表外部系統提供的功能。很類似DIP原則，但有點出入。
2. 在構造函數中注入依賴：不要在內部創建或直接依賴具體實現
3. 為每個外部依賴提供Mock/Stub實現：用於測試
4. 關注邊界情況：error handling、no data、timeout等
5. 驗證交互次數：確保外部系統被調用適當次數，驗證 SUT與DOC之間的交互行為
6. 分離配置與使用：連接字符串、API key 等配置應該可外部提供
7. 考慮使用測試容器：對於需要真實測試的場景，可使用Docker測試容器（或者使用 TestContainer）
8. 不信任外部系統：總是對外部數據進行驗證

通過依賴注入，我們不僅可以更容易地測試與外部系統交互的代碼，還可以在不修改業務邏輯的情況下，切換不同的實現（例如從PostgreSQL切換到MongoDB）。

# DI 管理套件
在實際的大型應用中，手動管理所有依賴注入會變得非常繁瑣，尤其是當依賴關係複雜時。**DI 容器**能幫助自動化這個過程，**管理物件生命週期**和**注入方式**。

## DI 容器的關鍵功能
DI 容器主要解決以下問題：

- 自動管理依賴關係：自動解析和注入複雜的依賴樹
- 生命週期管理：控制被注入物件的作用域和生命週期
- 集中配置：提供集中配置依賴關係的方式
- 減少樣板程式碼：減少創建和注入依賴的重複程式碼。能回憶上一段 `MockUserRepository`，可能有各種 Mock Repository 要自己寫。

## 依賴物件的生命週期
DI 容器通常支持多種生命週期模式：

- 單例 (Singleton)：整個應用中只有一個實例
  - 適用：資料庫連接池、全局配置、快取管理器等
- 瞬時 (Transient)：每次請求都創建新實例
  - 適用：非共享資源、無狀態的服務等
- 作用域 (Scoped)：在特定範圍內共享同一實例
  - 適用：HTTP 請求範圍內的服務、事務範圍等


## Uber fx
[Uber fx](https://github.com/uber-go/fx) 是一個基於依賴注入的 Go 語言應用框架，專注
於構建模塊化、可測試的應用。


特點
- 基於建構函數的依賴注入：通過分析函數參數和返回值自動解析依賴
- 生命週期管理：提供 OnStart 和 OnStop hook event 管理資源
- Modulize：支持將應用分解為多個可重用模塊
- 非侵入式設計：與標準 Go 程式碼配合良好

示例
```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net/http"
    
    _ "github.com/lib/pq"
    "go.uber.org/fx"
)

// 資料庫連接提供者（單例）
func NewDatabase() (*sql.DB, error) {
    return sql.Open("postgres", "postgres://user:password@localhost/mydb?sslmode=disable")
}

// 用戶倉庫
type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) GetUser(id int) (string, error) {
    var name string
    err := r.db.QueryRow("SELECT name FROM users WHERE id = $1", id).Scan(&name)
    return name, err
}

// 用戶服務
type UserService struct {
    repo *UserRepository
}

func NewUserService(repo *UserRepository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) GreetUser(id int) (string, error) {
    name, err := s.repo.GetUser(id)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("Hello, %s", name), nil
}

// HTTP 處理器（每個請求都是新的）
type UserHandler struct {
    service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
    return &UserHandler{service: service}
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    id := 1 // 從 URL 解析，這裡簡化處理
    greeting, err := h.service.GreetUser(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    fmt.Fprintln(w, greeting)
}

// HTTP 服務器
type Server struct {
    handler *UserHandler
}

func NewServer(handler *UserHandler) *Server {
    return &Server{handler: handler}
}

func (s *Server) Start(lifecycle fx.Lifecycle) {
    server := &http.Server{
        Addr:    ":8080",
        Handler: s.handler,
    }
    
    lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            // 啟動 HTTP 服務器
            go func() {
                log.Println("Starting HTTP server at :8080")
                if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                    log.Printf("HTTP server error: %v", err)
                }
            }()
            return nil
        },
        OnStop: func(ctx context.Context) error {
            // 關閉 HTTP 服務器
            log.Println("Stopping HTTP server")
            return server.Shutdown(ctx)
        },
    })
}

func main() {
    app := fx.New(
        // 註冊所有依賴
        fx.Provide(
            NewDatabase,        // 單例：資料庫連接
            NewUserRepository,  // 單例：依賴數據庫
            NewUserService,     // 單例：依賴數據庫和倉庫
            NewUserHandler,     // 單例：依賴服務
            NewServer,          // 單例：依賴處理器
        ),
        // 註冊需要啟動的組件
        fx.Invoke(func(server *Server) {}),
    )
    
    // 啟動應用並阻塞直到收到終止信號
    app.Run()
}
```

這個示例中的生命週期管理︰
1. 資料庫連接：作為單例注入，整個應用共享
2. 儲存庫和服務：也是單例，由fx管理生命週期
3. HTTP 處理器：可以是單例或每個請求創建（取決於需求）
4. 啟動和關閉hook：通過 lifecycle.Append 註冊

## Google Wire
[Google Wire](https://github.com/google/wire) 是一個編譯時依賴注入工具，與傳統的運行時DI容器不同，它在編譯時生成依賴注入程式碼。

**示例**
首先，我們定義相同的組件：
```go
// wire.go
//+build wireinject

package main

import (
    "github.com/google/wire"
)

// InitializeServer 是一個由 Wire 生成的函數，用於創建 Server 及其所有依賴
func InitializeServer() (*Server, error) {
    wire.Build(
        NewDatabase,
        NewUserRepository,
        NewUserService,
        NewUserHandler,
        NewServer,
    )
    return nil, nil // 返回值會被 Wire 生成的程式碼替換
}
```

執行 wire 命令後會生成：
```go
// wire_gen.go
// Code generated by Wire. DO NOT EDIT.

package main

import (
    "github.com/google/wire"
)

// InitializeServer is a function generated by Wire that creates a Server and its dependencies.
func InitializeServer() (*Server, error) {
    db, err := NewDatabase()
    if err != nil {
        return nil, err
    }
    userRepository := NewUserRepository(db)
    userService := NewUserService(userRepository)
    userHandler := NewUserHandler(userService)
    server := NewServer(userHandler)
    return server, nil
}
```

**Wire 的生命週期管理**
Wire 本身不負責生命週期管理，它只生成實例化的程式碼。生命週期管理需要在應用程式中自行處理：

1. 資源釋放：需要自行處理對象的釋放（如關閉數據庫連接）
2. 啟動/停止：需要自行管理組件的啟動和停止流程
3. 範圍管理：需要自行實現作用域範圍的管理

# 結論
依賴注入容器和工具為管理複雜的依賴關係提供了強大的支持，但選擇哪一種取決於項目的具體需求：

- 小型項目（POC or SideProejct）：可能直接使用手動依賴注入即可
- 中型項目：Google Wire 提供良好的平衡，不增加運行時開銷
- 大型項目：Uber fx 提供更完整的功能，包括生命週期管理

無論選擇哪種方式，核心的依賴注入原則都不變：將依賴的創建與使用分離，提高代碼的可測試性和靈活性。