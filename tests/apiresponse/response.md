# API Response Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：4／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[apiresponse/response_test.go](../../apiresponse/response_test.go)。
- **正式程式**：[apiresponse/response.go](../../apiresponse/response.go)。
- **案例數量**：3 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：JSON 回應、JSON 編碼失敗的日誌處理，以及標準錯誤回應封裝。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：使用 httptest.ResponseRecorder 與 bytes.Buffer；編碼錯誤案例使用 t.Cleanup 還原標準 log 輸出，不平行執行。所有案例皆為 P1。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `apiresponse/response_test.go` | 3 | 3 | 0 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-APR-001 | P1 | responsePayload={status: ok}，responseRecorder 為空。 | 呼叫 WriteJSON，指定 202 Accepted。 | 202；Content-Type=application/json；Body 可解碼且 status=ok。 | `TestWriteJSON_Success` |
| TC-APR-002 | P1 | unsupportedPayload 為 chan int；以 logOutput 收集日誌並註冊清理還原。 | 呼叫 WriteJSON，指定 200 OK。 | 日誌包含 encode response: 與 unsupported type；HTTP 狀態維持 200、Body 為空。 | `TestWriteJSON_LogsEncodingFailure` |
| TC-APR-003 | P1 | 錯誤代碼為 CodeAuthenticationRequired，訊息為 MessageAuthenticationRequired。 | 呼叫 WriteError，指定 401。 | 401；Body 可解碼為 ErrorBody，error.code 與 error.message 符合預期。 | `TestWriteError_EncodesErrorEnvelope` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test ./apiresponse -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `apiresponse/response.go` statement coverage | 100.0%（5／5 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

編碼失敗時回應狀態已先寫入，故現有測試預期 200 與空 Body，並驗證錯誤寫入日誌；不代表會另回傳 500 或標準錯誤 JSON。WriteError 案例驗證狀態與錯誤欄位，未獨立斷言其 Content-Type。
