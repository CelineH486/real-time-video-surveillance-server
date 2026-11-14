# Recording Service Cache Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：7／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[services/recording_service_test.go](../../services/recording_service_test.go)。
- **正式程式**：[services/recording_service.go](../../services/recording_service.go)。
- **案例數量**：3 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：expireCacheEntry 對相符、已替換與不存在快取項目的檔案清理行為。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：每個案例使用 t.TempDir 建立獨立暫存檔案與 RecordingService.cache，直接呼叫 expireCacheEntry；不等待實際計時器、不呼叫 MediaMTX。所有案例皆為 P2。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `services/recording_service_test.go` | 3 | 0 | 3 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-SREC-001 | P2 | cacheKey=expired-recording；建立 recording.mp4，cache 指向該檔，expiresAt 為 1 分鐘前。 | 呼叫 expireCacheEntry(cacheKey, cachedFilePath)。 | cache 項目移除；os.Stat 回傳 os.ErrNotExist。 | `TestExpireCacheEntry_RemovesMatchingFile` |
| TC-SREC-002 | P2 | cacheKey=replaced-recording 的當前檔案為 replacement.mp4，內容 replacement-content；過期回呼仍帶舊的 old.mp4 路徑。 | 呼叫 expireCacheEntry(cacheKey, oldFilePath)。 | 保留 cache 項目與 replacementFilePath；替換檔案內容完整。 | `TestExpireCacheEntry_PreservesReplacementFile` |
| TC-SREC-003 | P2 | cache 為空，但暫存目錄有 unregistered.mp4。 | 以不存在的 cacheKey 呼叫 expireCacheEntry。 | cache 仍為空；未登記的檔案仍存在。 | `TestExpireCacheEntry_MissingEntry` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test ./services -run '^TestExpireCacheEntry_' -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `services/recording_service.go` statement coverage | 88.1%（111／126 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

直接呼叫快取失效處理函式，未驗證計時器排程或並行下載。函式依 key 與 path 是否相符決定清理，案例不等待 expiresAt 自動到期。
