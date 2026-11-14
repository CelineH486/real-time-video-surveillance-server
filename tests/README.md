# Test Case Index

本索引依序整理 8 個測試檔的獨立 testcase，共 **70 個 `Test` 函式**。每份文件採用 AAA 格式，逐筆列出前置條件、操作、主要預期結果及實際測試函式。

文件日期：2026-11-13。

## 閱讀順序

| 順序 | 模組 | Testcase 文件 | 對應測試檔 | 案例數 |
| ---: | :--- | :--- | :--- | ---: |
| 1 | Main | [main/main.md](main/main.md) | [main_test.go](../main_test.go) | 18 |
| 2 | Routes | [routes/routes.md](routes/routes.md) | [routes/routes_test.go](../routes/routes_test.go) | 9 |
| 3 | Database | [db/db.md](db/db.md) | [db/db_test.go](../db/db_test.go) | 26 |
| 4 | API Response | [apiresponse/response.md](apiresponse/response.md) | [apiresponse/response_test.go](../apiresponse/response_test.go) | 3 |
| 5 | Authentication Service | [services/auth_service.md](services/auth_service.md) | [services/auth_service_test.go](../services/auth_service_test.go) | 3 |
| 6 | Location Service | [services/location_service.md](services/location_service.md) | [services/location_service_test.go](../services/location_service_test.go) | 6 |
| 7 | Recording Service Cache | [services/recording_service.md](services/recording_service.md) | [services/recording_service_test.go](../services/recording_service_test.go) | 3 |
| 8 | Status Service | [services/status_service.md](services/status_service.md) | [services/status_service_test.go](../services/status_service_test.go) | 2 |
| **合計** | **8 個模組** | **8 份獨立文件** | **8 個測試檔** | **70** |

既有 Controller 案例另見 [controllers/api.md](controllers/api.md)，其中的 119 個案例不計入上表。

## 文件與程式位置

- testcase 文件依模組放在 `tests` 的對應子資料夾。
- 上表連結的 `*_test.go` 是本索引的計數依據；同名但位於 `tests/services` 或 `tests/db` 的既有 Go 測試檔不計入這 70 個案例。
- 每份文件一個案例 ID 對應一個獨立 `Test` 函式，輔助函式不計為 testcase。
- 各文件只列既有測試實際驗證的主要結果，未宣稱所有輸入邊界或端到端行為均已測試。

## 整體測試與最終結果

從專案根目錄執行完整 Go 測試，並量測所有 Go package 的 statement coverage：

```powershell
go test -count=1 '-coverpkg=./...' '-coverprofile=coverage/overall-coverage.out' ./...
go tool cover '-func=coverage/overall-coverage.out'
```

查看 HTML 明細：

```powershell
go tool cover '-html=coverage/overall-coverage.out'
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近完整驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本索引對應案例數 | 70 |
| 本索引對應測試 | PASS |
| 完整專案測試 | PASS |
| 整體 statement coverage | **80.3%（772／961 statements）** |
| 未覆蓋 statements | 189 |

整體覆蓋率取合併報告的 `total`，以已覆蓋程式敘述數除以全部程式敘述數計算，不是各檔案或各 package 百分比的算術平均。各獨立文件的檔案覆蓋率也採相同合併結果。

上述指令未啟用 `e2e` build tag；結果不包含需要啟動正式 API 與 PostgreSQL 的 E2E 驗證。測試執行紀錄日期與文件日期分開標示。
