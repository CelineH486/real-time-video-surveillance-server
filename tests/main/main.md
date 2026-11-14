# Main Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：1／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[main_test.go](../../main_test.go)。
- **正式程式**：[main.go](../../main.go)。
- **案例數量**：18 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：環境變數讀取、CSV 設定解析、開發來源辨識與 CORS middleware；不啟動 main()、HTTP server、UDP server 或正式資料庫。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：使用 t.Setenv 隔離環境變數，使用 httptest 建立記憶體請求與 responseRecorder；所有案例皆為 P1。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `main_test.go` | 18 | 18 | 0 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-MAIN-001 | P1 | SURVEILLANCE_TEST_SETTING 設為前後含空白的 configured-value，fallback 為 fallback-value。 | 呼叫 env(settingName, fallback)。 | 回傳去除前後空白的 configured-value。 | `TestEnv_UsesConfiguredValue` |
| TC-MAIN-002 | P1 | SURVEILLANCE_TEST_SETTING 設為空字串。 | 呼叫 env，fallback=fallback-value。 | 回傳 fallback-value。 | `TestEnv_UsesFallbackForEmptyValue` |
| TC-MAIN-003 | P1 | SURVEILLANCE_TEST_SETTING 只包含空白與 tab。 | 呼叫 env，fallback=fallback-value。 | 回傳 fallback-value。 | `TestEnv_UsesFallbackForWhitespace` |
| TC-MAIN-004 | P1 | configuredOrigins 包含兩個不同來源、前後空白及重複的 first.example。 | 呼叫 splitCSV。 | map 只有 https://first.example 與 https://second.example，值皆為 true。 | `TestSplitCSV_TrimsAndDeduplicatesValues` |
| TC-MAIN-005 | P1 | configuredOrigins=, ,https://app.example,,。 | 呼叫 splitCSV。 | 只保留 https://app.example，忽略空項目。 | `TestSplitCSV_IgnoresEmptyValues` |
| TC-MAIN-006 | P1 | configuredOrigins 為空字串。 | 呼叫 splitCSV。 | 回傳非 nil 的空 map。 | `TestSplitCSV_EmptyInput` |
| TC-MAIN-007 | P1 | requestOrigin=http://localhost:5173。 | 呼叫 isLocalDevelopmentOrigin。 | true。 | `TestIsLocalDevelopmentOrigin_Localhost` |
| TC-MAIN-008 | P1 | requestOrigin=https://127.0.0.1:3000。 | 呼叫 isLocalDevelopmentOrigin。 | true。 | `TestIsLocalDevelopmentOrigin_IPv4Loopback` |
| TC-MAIN-009 | P1 | requestOrigin=http://[::1]:8080。 | 呼叫 isLocalDevelopmentOrigin。 | true。 | `TestIsLocalDevelopmentOrigin_IPv6Loopback` |
| TC-MAIN-010 | P1 | requestOrigin=https://localhost.attacker.example。 | 呼叫 isLocalDevelopmentOrigin。 | false；不將相似網域視為本機來源。 | `TestIsLocalDevelopmentOrigin_RejectsLookalikeHost` |
| TC-MAIN-011 | P1 | requestOrigin=ftp://localhost。 | 呼叫 isLocalDevelopmentOrigin。 | false；拒絕非 HTTP／HTTPS scheme。 | `TestIsLocalDevelopmentOrigin_RejectsUnsupportedScheme` |
| TC-MAIN-012 | P1 | requestOrigin=://invalid-origin。 | 呼叫 isLocalDevelopmentOrigin。 | false；拒絕解析失敗的 URL。 | `TestIsLocalDevelopmentOrigin_RejectsMalformedURL` |
| TC-MAIN-013 | P1 | 允許 https://app.example；請求 Origin 相同；protectedHandler 記錄呼叫並回傳 202。 | 以 corsHandler.ServeHTTP 執行 GET /api/trucks。 | handler 被呼叫；202；Access-Control-Allow-Origin 為設定來源；Vary=Origin。 | `TestWithCORS_ConfiguredOrigin` |
| TC-MAIN-014 | P1 | allowedOrigins=*；requestOrigin=https://external.example。 | 以 corsHandler 執行 GET 請求。 | Access-Control-Allow-Origin 回傳實際 requestOrigin。 | `TestWithCORS_WildcardOrigin` |
| TC-MAIN-015 | P1 | 只設定 https://app.example；requestOrigin=http://localhost:5173。 | 以 corsHandler 執行 GET 請求。 | Access-Control-Allow-Origin 為本機開發來源。 | `TestWithCORS_LocalDevelopmentOrigin` |
| TC-MAIN-016 | P1 | 只允許 https://app.example；請求來自 https://untrusted.example。 | 以 corsHandler 執行一般 GET 請求。 | 仍呼叫 protectedHandler，但不設定 Access-Control-Allow-Origin。 | `TestWithCORS_DisallowedOrigin` |
| TC-MAIN-017 | P1 | allowedOrigins=*；請求未提供 Origin header。 | 以 corsHandler 執行 GET 請求。 | 呼叫 protectedHandler，不設定 Access-Control-Allow-Origin。 | `TestWithCORS_NoOriginHeader` |
| TC-MAIN-018 | P1 | 允許 https://app.example；建立帶該 Origin 的 OPTIONS /api/trucks 請求。 | 以 corsHandler 執行 preflightRequest。 | 204、Body 為空且不呼叫 protectedHandler；允許的方法為 GET, POST, PATCH, DELETE, OPTIONS；允許的 headers 為 Authorization, Content-Type。 | `TestWithCORS_PreflightRequest` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test . -run '^(TestEnv_|TestSplitCSV_|TestIsLocalDevelopmentOrigin_|TestWithCORS_)' -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `main.go` statement coverage | 27.4%（32／117 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

未執行 main() 的啟動流程，未驗證資料庫連線、背景服務啟動或 HTTP listener；main.go 的檔案覆蓋率不能當作這四個輔助函式各自的覆蓋率。
