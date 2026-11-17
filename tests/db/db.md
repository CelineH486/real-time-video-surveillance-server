# Database Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **文件日期**：2026-11-13。
- **閱讀順序**：3／8；[返回測試案例索引](../README.md)。
- **測試原始檔**：[db/db_test.go](../../db/db_test.go)。
- **正式程式**：[db/db.go](../../db/db.go)。
- **案例數量**：26 個獨立 `Test` 函式；每個函式對應一筆案例。
- **涵蓋範圍**：資料庫連線字串失敗、攝影機及卡車查詢、狀態更新、錄影查詢、API Token 查詢與定位交易失敗處理。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立測試資料、請求、mock、環境變數或暫存檔案。 |
| Act | 執行操作 | 呼叫被測函式或已設定的 handler。 |
| Assert | 主要預期結果 | 驗證可觀察回應、資料、錯誤、SQL 預期或檔案狀態。 |

- **P1**：純邏輯或記憶體 HTTP 測試，不需要 SQL mock 或實體檔案 fixture。
- **P2**：使用 SQL mock 或隔離的暫存檔案；階段代表依賴分類，不代表執行順序。
- **測試方式**：除 Connect 案例使用無效連線字串外，其餘皆使用 sqlmock，不連線正式 PostgreSQL。newDatabaseTestConnection 在清理階段驗證 SQL 預期並關閉連線。
- **命名原則**：變數以用途命名，例如 `databaseConnection`、`responseRecorder`、`expectedDatabaseError`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 測試檔 | 獨立 Test 數 | P1 | P2 |
| :--- | ---: | ---: | ---: |
| `db/db_test.go` | 26 | 1 | 25 |

## 4. 測試案例

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-DB-001 | P1 | DATABASE_URL=invalid connection string。 | 呼叫 Connect。 | databaseConnection 為 nil，connectionError 非 nil。 | `TestConnect_InvalidConnectionString` |
| TC-DB-002 | P2 | 查詢回傳 cam01／truck001／Front Camera／online 與 cam02／truck001／Rear Camera／offline。 | 呼叫 GetCameras。 | 回傳兩筆完整對應的 map，錯誤為 nil，資料列關閉。 | `TestGetCameras_Success` |
| TC-DB-003 | P2 | 攝影機查詢回傳 expectedDatabaseError。 | 呼叫 GetCameras。 | 回傳 nil slice 與原始查詢錯誤。 | `TestGetCameras_QueryError` |
| TC-DB-004 | P2 | 攝影機資料的 camera_id 為 NULL。 | 呼叫 GetCameras。 | 回傳 nil slice 與非 nil 掃描錯誤，資料列關閉。 | `TestGetCameras_ScanError` |
| TC-DB-005 | P2 | 卡車查詢回傳 truck001／ABC-1234／online。 | 呼叫 GetTrucks。 | 回傳包含 truckId、plateNo、status 的預期 slice，錯誤為 nil，資料列關閉。 | `TestGetTrucks_Success` |
| TC-DB-006 | P2 | 卡車查詢回傳 expectedDatabaseError。 | 呼叫 GetTrucks。 | 回傳 nil slice 與原始查詢錯誤。 | `TestGetTrucks_QueryError` |
| TC-DB-007 | P2 | 卡車資料的 plate_no 為 NULL。 | 呼叫 GetTrucks。 | 回傳 nil slice 與非 nil 掃描錯誤，資料列關閉。 | `TestGetTrucks_ScanError` |
| TC-DB-008 | P2 | truckID=truck001、newTruckStatus=offline；UPDATE 預期參數為 status、truckID。 | 呼叫 UpdateTruckStatus。 | SQL 預期完成且 updateError 為 nil。 | `TestUpdateTruckStatus_Success` |
| TC-DB-009 | P2 | 更新 truck001 為 offline 時 SQL 回傳 expectedDatabaseError。 | 呼叫 UpdateTruckStatus。 | 回傳原始資料庫錯誤。 | `TestUpdateTruckStatus_DatabaseError` |
| TC-DB-010 | P2 | cameraID=cam01、truckID=truck001、newCameraStatus=online；SQL 包含 last_seen_at=NOW()。 | 呼叫 UpdateCameraStatus。 | SQL 的參數順序為 status、cameraID、truckID；回傳 nil。 | `TestUpdateCameraStatus_Success` |
| TC-DB-011 | P2 | 更新 cam01／truck001 為 online 時 SQL 失敗。 | 呼叫 UpdateCameraStatus。 | 回傳 expectedDatabaseError。 | `TestUpdateCameraStatus_DatabaseError` |
| TC-DB-012 | P2 | offlineTimeout=15s；SQL 篩選過期 last_seen_at，RowsAffected=3。 | 呼叫 MarkStaleCamerasOffline。 | 回傳 updatedCameraCount=3 與 nil。 | `TestMarkStaleCamerasOffline_ReturnsAffectedCount` |
| TC-DB-013 | P2 | 將過期攝影機設為 offline 的 SQL 回傳錯誤。 | 呼叫 MarkStaleCamerasOffline。 | 回傳數量 0 與 expectedDatabaseError。 | `TestMarkStaleCamerasOffline_DatabaseError` |
| TC-DB-014 | P2 | SQL 執行成功，但 sqlmock.NewErrorResult 讓 RowsAffected 回傳錯誤。 | 呼叫 MarkStaleCamerasOffline。 | 回傳數量 0 與 expectedResultError。 | `TestMarkStaleCamerasOffline_RowsAffectedError` |
| TC-DB-015 | P2 | 設定依 cameras 的 online 狀態更新 trucks 的 SQL 預期，執行成功。 | 呼叫 RefreshTruckStatusesFromCameras。 | SQL 預期完成，refreshError 為 nil。 | `TestRefreshTruckStatusesFromCameras_Success` |
| TC-DB-016 | P2 | 卡車狀態更新 SQL 回傳 expectedDatabaseError。 | 呼叫 RefreshTruckStatusesFromCameras。 | 回傳原始資料庫錯誤。 | `TestRefreshTruckStatusesFromCameras_DatabaseError` |
| TC-DB-017 | P2 | 查詢 truck001／cam01；回傳 id=42、開始時間 2026-11-11T08:00:00Z、結束時間晚 1 分鐘、路徑 recordings/clip.mp4、4096 bytes、complete。 | 呼叫 GetRecordings。 | 回傳與 expectedRecordings 完全相同的 models.Recording slice，錯誤為 nil，資料列關閉。 | `TestGetRecordings_Success` |
| TC-DB-018 | P2 | truckID=truck001、cameraID 為空；SQL 沒有回傳資料列。 | 呼叫 GetRecordings。 | 回傳非 nil 的空 slice 與 nil。 | `TestGetRecordings_EmptyResult` |
| TC-DB-019 | P2 | truck001／cam01 的錄影查詢回傳 expectedDatabaseError。 | 呼叫 GetRecordings。 | 回傳 nil slice 與原始資料庫錯誤。 | `TestGetRecordings_QueryError` |
| TC-DB-020 | P2 | recording_id=invalid-id，無法轉為 int64。 | 呼叫 GetRecordings。 | 回傳 nil slice 與非 nil 掃描錯誤，資料列關閉。 | `TestGetRecordings_ScanError` |
| TC-DB-021 | P2 | 以 RowError(0, expectedReadError) 模擬讀取資料列失敗。 | 呼叫 GetRecordings。 | queryError 為 expectedReadError。 | `TestGetRecordings_RowIterationError` |
| TC-DB-022 | P2 | 固定 lookupTime；missing-token 的 hash 查詢回傳 sql.ErrNoRows。 | 呼叫 UserByAPIToken。 | 空 AuthenticatedUser、userFound=false、queryError=nil。 | `TestUserByAPIToken_NotFound` |
| TC-DB-023 | P2 | 固定 lookupTime；test-token 的 hash 查詢回傳 expectedDatabaseError。 | 呼叫 UserByAPIToken。 | 空 AuthenticatedUser、userFound=false、queryError 為原始錯誤。 | `TestUserByAPIToken_DatabaseError` |
| TC-DB-024 | P2 | 定位為 latitude=25、longitude=121、speedKmh=10、fixQuality=1；固定 recordedAt；Begin 成功，truck_locations INSERT 失敗，預期 Rollback。 | 呼叫 SaveTruckLocation。 | 回傳歷史寫入的原始錯誤；SQL 預期確認執行 Rollback，未繼續寫入狀態表。 | `TestSaveTruckLocation_HistoryInsertErrorRollsBack` |
| TC-DB-025 | P2 | 同一組定位與固定時間；Begin、歷史表 INSERT 成功；truck_location_state INSERT 失敗，預期 Rollback。 | 呼叫 SaveTruckLocation。 | 回傳狀態表寫入的原始錯誤；SQL 預期確認執行 Rollback。 | `TestSaveTruckLocation_StateInsertErrorRollsBack` |
| TC-DB-026 | P2 | 同一組定位與固定時間；Begin、兩次 INSERT 成功；Commit 回傳 expectedDatabaseError。 | 呼叫 SaveTruckLocation。 | 回傳 Commit 的原始錯誤。 | `TestSaveTruckLocation_CommitError` |

## 5. 執行方式與最終結果

從專案根目錄執行本文件對應的測試：

```powershell
go test ./db -count=1
```

| 項目 | 最終結果 |
| :--- | :--- |
| 最近驗證日期 | 2026-11-11 |
| Go 版本 | 1.27.1 |
| 本文件對應測試 | PASS |
| 完整專案測試 | PASS |
| `db/db.go` statement coverage | 95.2%（199／209 statements） |

覆蓋率為完整專案測試合併後的原始檔結果，包含其他測試對該檔案的覆蓋，不代表單獨執行本文件案例的覆蓋率。整體測試指令與結果見[測試案例索引](../README.md#整體測試與最終結果)。

## 6. 驗證範圍

sqlmock 驗證查詢參數、回傳資料、錯誤傳遞與交易順序，不執行 PostgreSQL SQL，也不驗證 migration 或真正連線成功的流程。Commit 失敗案例只驗證錯誤傳遞；兩個 INSERT 失敗案例另有 Rollback 預期。
