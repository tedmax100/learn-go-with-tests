Token Monitor
-------------

# 模組原本功能

- TokenMonitor類別 - 核心监控服务
- ServiceParams結構 - Fx容器依賴注入參數
- entity.NotificationEvent - 通知事件
- slack.SlackClientInterface - 外部Slack客户端介面

主要功能：

- 周期性執行token檢查
- 周期性執行oracle accept檢查
- 處理通知事件
- 通知模板處理和Slack發送通知

```
+-----------------------+          +---------------------+
|    TokenMonitor       |          | SlackClientInterface|
+-----------------------+          +---------------------+
| - notificationChan    |<-------->| + SendNotification()|
| - slackClient         |--------->+---------------------+
| - ctx, cancel         |
| - ticker              |          +---------------------+
| - checkTokensFunc     |<---------| NotificationEvent   |
| - checkInterval       |          +---------------------+
| - acceptTicker        |          | - Type              |
| - checkOracleAcceptFn |          | - OtherData         |
| - acceptCheckInterval |          +---------------------+
+-----------------------+
| + SetCheckTokensFunc()|
| + SetCheckInterval()  |
| + SetOracleAcceptFn() |
| + SetAcceptInterval() |
| + Run()               |
| + Stop()              |
| + processNotification()|
| + processContractVerify()|
+-----------------------+
```

# TokenMonitor 簡化版本

```
+-----------------------+
|    TokenMonitor       |
+-----------------------+
| - notificationChan    |<----- 接收通知的 channel
| - ticker              |<----- 定時器
| - checkFunc           |<----- 檢查函數
| - interval            |<----- 檢查間隔
| - ctx                 |
| - cancel              |<----- 取消函數
| - ProcessNotification |<----- 處理通知的函數
+-----------------------+
| + NewTokenMonitor()   |<----- constructor
| + SetCheckFunc()      |<----- 設置檢查函數
| + SetInterval()       |<----- 設置檢查間隔
| + Run()               |<----- 啟動監控
| + Stop()              |<----- 停止監控
+-----------------------+

```

## 類別說明

`TokenMonitor` 是一個簡化版的監控服務，主要功能包括：

1. **定期執行檢查函數**：通過內部timer，按設定的間隔時間執行用戶提供的檢查函數。
2. **處理通知事件**：從通知 channel 接收消息，並使用處理函數進行處理。
3. **生命週期管理**：提供啟動和停止功能，以控制監控服務的生命週期。

## 屬性說明

- `notificationChan`：接收通知的read only channel
- `ticker`：Timer，用於定期觸發檢查函數
- `checkFunc`：用戶提供的檢查函數，接收一個 context.Context 參數
- `interval`：檢查間隔時間
- `ctx` 和 `cancel`：用於控制服務生命週期的上下文和取消函數
- `ProcessNotification`：處理通知的函數，可由用戶自定義

## 方法說明

- `NewTokenMonitor()`：Constructor，創建一個新的 TokenMonitor instance
- `SetCheckFunc()`：設置要定期執行的檢查函數
- `SetInterval()`：設置檢查間隔時間
- `Run()`：啟動監控服務，開始定期執行檢查並處理通知
- `Stop()`：停止監控服務，釋放資源

## 工作流程

1. 創建 TokenMonitor Instance
2. 設置檢查函數和間隔時間
3. 調用 Run() 啟動服務
4. 服務會在背景執行以下操作：
   - 定期執行檢查函數
   - 處理從通知 channel 接收的事件
5. 當不再需要服務時，調用 Stop() 停止服務
