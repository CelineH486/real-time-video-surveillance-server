# Location Service Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：6／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[services/location_service_test.go](../../services/location_service_test.go)。
- **正式程式**：[services/location_service.go](../../services/location_service.go)。
- **案例數量**：6 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：GPS 選填欄位、未來時間界限與第一筆停止定位的 stop anchor 建立。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：validationTime／recordedAt 固定為 2026-11-11T08:00:00Z；基本定位為有效座標與 fixQuality=1，每個驗證案例只改變一個目標條件。所有案例皆為 P1。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `services/location_service_test.go` | 6 | 6 | 0 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-SLOC-001 | P1 | 有效定位的 headingDegrees=360。 | 呼叫 ValidateLocation。 | 回傳 ErrInvalidLocation。 | `TestValidateLocation_RejectsInvalidHeading` |
| TC-SLOC-002 | P1 | 有效定位的 accuracyMeters=-1。 | 呼叫 ValidateLocation。 | 回傳 ErrInvalidLocation。 | `TestValidateLocation_RejectsNegativeAccuracy` |
| TC-SLOC-003 | P1 | 有效定位的 satelliteCount=-1。 | 呼叫 ValidateLocation。 | 回傳 ErrInvalidLocation。 | `TestValidateLocation_RejectsNegativeSatelliteCount` |
| TC-SLOC-004 | P1 | recordedAt 為 validationTime 加 5 分鐘又 1 秒。 | 呼叫 ValidateLocation。 | 回傳 ErrInvalidLocation。 | `TestValidateLocation_RejectsTimestampBeyondFutureLimit` |
| TC-SLOC-005 | P1 | recordedAt 恰好為 validationTime 加 5 分鐘。 | 呼叫 ValidateLocation。 | 回傳 nil，接受時間界限。 | `TestValidateLocation_AcceptsTimestampAtFutureLimit` |
| TC-SLOC-006 | P1 | 沒有上一筆定位；latitude=25.033、longitude=121.5654、speedKmh=0.5。 | 呼叫 StopState(nil, locationInput, recordedAt)。 | anchorLatitude、anchorLongitude、stoppedSince 均非 nil；座標符合輸入，stoppedSince 等於 recordedAt。 | `TestStopState_FirstStoppedLocationCreatesAnchor` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test ./services -run '^(TestValidateLocation_|TestStopState_FirstStoppedLocationCreatesAnchor$)' -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `services/location_service.go` statement coverage | 100.0%（30／30 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

只列本測試檔的六個情境；座標、速度、fixQuality 及停止狀態的其他情境由既有測試涵蓋。測試不連線資料庫，也不驗證控制器的 HTTP 回應。
