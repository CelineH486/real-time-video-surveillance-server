# Routes Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：2／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[routes/routes_test.go](../../routes/routes_test.go)。
- **正式程式**：[routes/routes.go](../../routes/routes.go)。
- **案例數量**：9 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：以正式 ServeMux 驗證頁面跳轉、靜態資源、健康檢查、認證 middleware 與登入路由。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：newRouteTestHandler 建立各控制器與真實認證 middleware，資料庫依賴使用 nil；fstest.MapFS 提供記憶體靜態檔案，不需本機 HTTP server。所有案例皆為 P1。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `routes/routes_test.go` | 9 | 9 | 0 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-ROUTE-001 | P1 | 建立正式 routeHandler 與 GET / 請求。 | 呼叫 routeHandler.ServeHTTP。 | 307 Temporary Redirect；Location=/login。 | `TestNew_RootRedirectsToLogin` |
| TC-ROUTE-002 | P1 | 建立 GET /missing-page 請求。 | 呼叫 routeHandler.ServeHTTP。 | 404 Not Found。 | `TestNew_UnknownPathReturnsNotFound` |
| TC-ROUTE-003 | P1 | 建立 GET /web 請求。 | 呼叫 routeHandler.ServeHTTP。 | 307 Temporary Redirect；Location=/web/。 | `TestNew_WebRedirectAddsTrailingSlash` |
| TC-ROUTE-004 | P1 | fstest.MapFS 的 index.html 內容為 `<!doctype html><title>Surveillance Login</title>`。 | 執行 GET /login。 | 200；Body 等於 indexHTML；Content-Type=text/html; charset=utf-8；Cache-Control=no-cache。 | `TestNew_LoginServesAppIndex` |
| TC-ROUTE-005 | P1 | 使用不含 index.html 的空 fstest.MapFS。 | 執行 GET /login。 | 500；Body 包含 Web application unavailable。 | `TestNew_LoginReturnsErrorWhenIndexIsMissing` |
| TC-ROUTE-006 | P1 | fstest.MapFS 的 app.js 內容為 console.log('surveillance');。 | 執行 GET /web/app.js。 | 200；Body 等於 assetContent，證明 /web/ 前綴正確移除。 | `TestNew_WebServesStaticAsset` |
| TC-ROUTE-007 | P1 | 建立未帶認證的 GET /health 請求。 | 呼叫 routeHandler.ServeHTTP。 | 200；Body 包含 status=ok 的 JSON 片段。 | `TestNew_HealthRouteIsPublic` |
| TC-ROUTE-008 | P1 | 建立沒有 Authorization 的 GET /api/trucks 請求。 | 呼叫 routeHandler.ServeHTTP。 | 401；Body 包含 missing_bearer_token。 | `TestNew_TruckRouteRequiresAuthentication` |
| TC-ROUTE-009 | P1 | 建立 POST /api/auth/login，Body 為不完整的 email JSON。 | 呼叫 routeHandler.ServeHTTP。 | 400；Body 包含 invalid_login。 | `TestNew_LoginRouteValidatesBody` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test ./routes -run '^TestNew_' -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `routes/routes.go` statement coverage | 100.0%（33／33 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

使用記憶體檔案系統與請求，不啟動實際 HTTP server。New 的 100% statement coverage 不代表所有註冊路由的業務情境均已逐一測試。
