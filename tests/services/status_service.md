# Status Service Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：8／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[services/status_service_test.go](../../services/status_service_test.go)。
- **正式程式**：[services/status_service.go](../../services/status_service.go)。
- **案例數量**：2 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：StatusService 建構與空 MediaMTX URL 停用監控。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：databaseConnection 為 nil，MediaMTX URL 使用 /// 驗證移除尾端斜線後為空的情況；不啟動 UDP listener 或週期性監控。所有案例皆為 P1。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `services/status_service_test.go` | 2 | 2 | 0 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-SSTAT-001 | P1 | databaseConnection 為 nil。 | 呼叫 NewStatusService。 | statusService 非 nil，database 等於傳入依賴。 | `TestNewStatusService` |
| TC-SSTAT-002 | P1 | StatusService 的資料庫為 nil；mediaMTXBaseURL=///。 | 呼叫 MonitorMediaMTX。 | 函式正常返回，database 仍為 nil。 | `TestMonitorMediaMTX_EmptyURLDisablesMonitoring` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test ./services -run '^(TestNewStatusService|TestMonitorMediaMTX_EmptyURLDisablesMonitoring)$' -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `services/status_service.go` statement coverage | 6.9%（4／58 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

只驗證建構及空 URL 分支，不涵蓋 ListenUDP、MonitorOffline 或有效 MediaMTX URL 的持續監控迴圈。空 URL 案例主要斷言函式返回及 database 仍為 nil，未使用網路攔截器。
