# Authentication Service Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：5／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[services/auth_service_test.go](../../services/auth_service_test.go)。
- **正式程式**：[services/auth_service.go](../../services/auth_service.go)。
- **案例數量**：3 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：認證資料庫失敗、未知 API Token，以及透過 WebSocket subprotocol 建立使用者 Context。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：使用 sqlmock 模擬 UserByAPIToken；認證查詢必須傳入 APITokenHash(apiToken) 與查詢時間。所有案例皆為 P2。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `services/auth_service_test.go` | 3 | 0 | 3 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-SAUTH-001 | P2 | 有效格式的 Bearer authentication-test-token；資料庫查詢回傳 authentication store unavailable。 | 執行認證 middleware。 | 503；protectedHandler 未執行；JSON 的 error.code=authentication_unavailable。 | `TestAuthMiddleware_DatabaseUnavailable` |
| TC-SAUTH-002 | P2 | Bearer unknown-test-token；資料庫查詢回傳 sql.ErrNoRows。 | 執行認證 middleware。 | 401；protectedHandler 未執行；JSON 的 error.code=invalid_bearer_token。 | `TestAuthMiddleware_UnknownToken` |
| TC-SAUTH-003 | P2 | 沒有 Authorization；Sec-WebSocket-Protocol=bearer, websocket-test-token；查詢回傳 user001／driver@example.com／Test Driver。 | 執行認證 middleware；下游 handler 讀取 CurrentUser。 | 204；userFoundInContext=true；authenticatedUser 與 expectedUser 相同。 | `TestAuthMiddleware_WebSocketTokenPopulatesUserContext` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test ./services -run '^(TestAuthMiddleware_DatabaseUnavailable|TestAuthMiddleware_UnknownToken|TestAuthMiddleware_WebSocketTokenPopulatesUserContext)$' -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `services/auth_service.go` statement coverage | 100.0%（28／28 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

WebSocket 案例只驗證 Sec-WebSocket-Protocol 的 Token 認證與 Context 建立，不執行 WebSocket 升級。其餘 Authorization header 格式等情境由其他既有測試涵蓋。
