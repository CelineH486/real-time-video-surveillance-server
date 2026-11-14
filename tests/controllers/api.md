# Controller API Test Cases

## 1. 文件範圍

- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）。
- **更新日期**：2026-11-10。
- **整理依據**：目前 `controllers/*_test.go` 的 10 個測試檔，以及 [正式路由](../../routes/routes.go)、控制器與相關服務的實作。
- **案例數量**：119 個獨立 `Test` 函式，每個函式對應一筆案例。
- **案例 ID**：採用 `TC-模組代碼-流水號`，每個 ID 對應一個測試函式。
- **計數範圍**：`tests/controllers/session_controller_test.go` 的 3 個公開格式驗證測試屬於另一個 package，不計入本文件的 119 個案例。

## 2. AAA 與測試約定

| 階段 | 文件欄位 | 內容 |
| :--- | :--- | :--- |
| Arrange | 前置條件／輸入 | 建立控制器、請求、測試資料、SQL 預期或本機測試伺服器。 |
| Act | 執行操作 | 呼叫被測函式或透過已設定的 handler 執行請求。 |
| Assert | 主要預期結果 | 驗證 HTTP 狀態、JSON、回傳值、Token、時間範圍或其他可觀察結果。 |

- **P1**：不需要 SQL mock 或本機 HTTP／WebSocket server 的測試。
- **P2**：使用 SQL mock 或本機 HTTP／WebSocket server 的測試；這是依賴分類，不代表執行先後順序。
- SQL 使用既有 `sqlmock` 套件；不連線正式 PostgreSQL。HTTP 使用 `httptest`，WebSocket 使用本機測試 server。
- 表中的 `List`、`Play` 等操作預設為該章節控制器的方法。正式路由只供 API 對照；多數測試直接呼叫控制器並設定 `PathValue`，不代表已驗證 `ServeMux` 的路由分派。
- 「未登入」表示請求沒有已驗證使用者 Context，即使有 Authorization header，直接呼叫控制器也不會自動完成登入驗證。
- 表中列測試已有的主要斷言。Authorization、Truck、Stream 與 Response 的共用錯誤斷言會檢查 HTTP 狀態、`application/json`、`error.code` 及完整英文 `error.message`；Session 的錯誤斷言則檢查狀態、代碼與 message 非空。
- 測試資料名稱以用途為主，例如 `requestedTruckID`、`authenticatedRequest`、`responseRecorder`、`streamTokenClaims`；錯誤訊息與程式註釋使用英文。

## 3. 案例總覽

| 模組 | 測試檔 | 獨立 Test 數 |
| :--- | :--- | ---: |
| CameraController | [camera_controller_test.go](../../controllers/camera_controller_test.go) | 7 |
| RecordingController | [recording_controller_test.go](../../controllers/recording_controller_test.go) | 41 |
| LocationController | [location_controller_test.go](../../controllers/location_controller_test.go) | 15 |
| SessionController | [session_controller_test.go](../../controllers/session_controller_test.go) | 31 |
| HealthController | [health_controller_test.go](../../controllers/health_controller_test.go) | 1 |
| TruckController | [truck_controller_test.go](../../controllers/truck_controller_test.go) | 5 |
| StreamController | [stream_controller_test.go](../../controllers/stream_controller_test.go) | 7 |
| MediaMTXController | [mediamtx_controller_test.go](../../controllers/mediamtx_controller_test.go) | 4 |
| Authorization | [authorization_test.go](../../controllers/authorization_test.go) | 6 |
| Response | [response_test.go](../../controllers/response_test.go) | 2 |
| **合計** | **10 個測試檔** | **119** |

## 4. CameraController

正式路由：`GET /api/cameras`、`GET /api/trucks/{truckID}/cameras`。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-CAM-001 | P1 | 資料庫與串流服務皆為 nil。 | 呼叫 `NewCameraController`。 | Controller 實例不為 nil | `TestNewCameraController` |
| TC-CAM-002 | P1 | 建立沒有已驗證使用者 Context 的 GET /api/cameras 請求。 | 執行 `List`。 | 401 Unauthorized | `TestCameraController_List_Unauthorized` |
| TC-CAM-003 | P1 | 設定 truckID=truck001，但請求沒有已驗證使用者 Context。 | 執行 `ListByTruck`。 | 401 Unauthorized | `TestCameraController_ListByTruck_Unauthorized` |
| TC-CAM-004 | P2 | Mock 認證成功；使用者的攝影機查詢回傳 cam01。 | 經認證 middleware 執行 `List`。 | 200 OK；回應包含 cam01，SQL 預期均被執行 | `TestCameraController_List_Success` |
| TC-CAM-005 | P2 | Mock 認證、卡車權限與 cam01 查詢成功；設定串流服務並記錄執行前後時間。 | 經認證 middleware 執行 `ListByTruck`。 | 200；一台 cam01；subUrl 正確；subToken 有效且綁定 truck001／cam01／sub；期限約 5 分鐘，容許 RFC3339 的秒級截斷；SQL 預期完成。 | `TestCameraController_ListByTruck_Success` |
| TC-CAM-006 | P2 | 使用者已驗證，但查詢可觀看攝影機的 SQL 回傳錯誤 | 執行 `List`。 | 500 Internal Server Error | `TestCameraController_List_DatabaseError` |
| TC-CAM-007 | P2 | 使用者有卡車權限，但該卡車攝影機查詢發生 SQL 錯誤 | 執行 `ListByTruck`。 | 500 Internal Server Error | `TestCameraController_ListByTruck_DatabaseError` |

## 5. RecordingController

正式路由：

- `GET /api/trucks/{truckID}/recordings?cameraId={cameraID}`：列出錄影。
- `POST /api/trucks/{truckID}/cameras/{cameraID}/recordings/play`：建立錄影播放資訊。
- `GET /api/trucks/{truckID}/cameras/{cameraID}/recordings/content`：讀取錄影內容。

共同輸入：`truck001`、`cam01`、`recordingStart=2026-10-20T08:00:00Z`。

Recording Token 使用 `SignRecordingAccess`／`ValidateRecordingAccess`，綁定卡車、攝影機、錄影開始時間、可播放時長與到期時間。`Content` 檢查請求是否落在授權視窗內。

到期規則為目前時間加上向上取整的播放秒數，再加 10 分鐘緩衝；`durationSeconds=30.5` 對應約 10 分 31 秒。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-REC-001 | P1 | 資料庫、錄影服務與串流服務皆為 nil。 | 呼叫 `NewRecordingController`。 | Controller 實例不為 nil | `TestNewRecordingController` |
| TC-REC-002 | P1 | 錄影清單請求沒有已驗證使用者 Context。 | 執行 `List`。 | 401 Unauthorized | `TestRecordingController_List_Unauthorized` |
| TC-REC-003 | P1 | 錄影播放請求沒有已驗證使用者 Context。 | 執行 `Play`。 | 401 Unauthorized | `TestRecordingController_Play_Unauthorized` |
| TC-REC-004 | P2 | 使用者已登入但沒有指定卡車存取權限 | 執行 `List`。 | 403 Forbidden | `TestRecordingController_List_AccessErrors_Forbidden` |
| TC-REC-005 | P2 | 卡車權限查詢發生 SQL 錯誤 | 執行 `List`。 | 503 Service Unavailable | `TestRecordingController_List_AccessErrors_AccessDatabaseError` |
| TC-REC-006 | P2 | 認證與卡車權限通過；請求缺少 cameraId。 | 執行 `List`。 | 400 Bad Request | `TestRecordingController_List_Validation_MissingCameraID` |
| TC-REC-007 | P2 | 認證與卡車權限通過；cameraId=cam01 的存在性查詢回傳 false。 | 執行 `List`。 | 404 Not Found | `TestRecordingController_List_Validation_CameraNotFound` |
| TC-REC-008 | P2 | 認證與卡車權限通過；攝影機存在性查詢回傳 SQL 錯誤。 | 執行 `List`。 | 500 Internal Server Error | `TestRecordingController_List_Validation_CameraLookupError` |
| TC-REC-009 | P2 | 認證、權限與攝影機檢查通過；start=bad-time。 | 執行 `List`。 | 400 Bad Request | `TestRecordingController_List_Validation_InvalidStart` |
| TC-REC-010 | P2 | 認證、權限與攝影機檢查通過；end=bad-time。 | 執行 `List`。 | 400 Bad Request | `TestRecordingController_List_Validation_InvalidEnd` |
| TC-REC-011 | P2 | 認證、權限與攝影機檢查通過；本機 MediaMTX /list 回傳一筆長度 30.5 秒的錄影。 | 以合法 start 與 end 執行 `List`。 | 200；上游收到 GET /list、path、segments=true、start、end 與 Bearer header；回應一筆正確開始時間及 durationSeconds=30.5。 | `TestRecordingController_List_Success` |
| TC-REC-012 | P2 | 認證、權限與攝影機檢查通過；本機 MediaMTX /list 回傳 503。 | 執行 `List`。 | 502 Bad Gateway | `TestRecordingController_List_UpstreamError_MediaMTXUnavailable` |
| TC-REC-013 | P2 | 認證、權限與攝影機檢查通過；本機 MediaMTX /list 回傳 200 與 not-json。 | 執行 `List`。 | 502 Bad Gateway | `TestRecordingController_List_UpstreamError_InvalidMediaMTXJSON` |
| TC-REC-014 | P2 | 認證與卡車權限通過；請求本文不是合法 JSON。 | 執行 `Play`。 | 400 Bad Request | `TestRecordingController_Play_Validation_InvalidJSON` |
| TC-REC-015 | P2 | 認證與卡車權限通過；缺少 start。 | 執行 `Play`。 | 400 Bad Request | `TestRecordingController_Play_Validation_MissingStart` |
| TC-REC-016 | P2 | 認證與卡車權限通過；start 格式無效。 | 執行 `Play`。 | 400 Bad Request | `TestRecordingController_Play_Validation_InvalidStart` |
| TC-REC-017 | P2 | 認證與卡車權限通過；durationSeconds = 0。 | 執行 `Play`。 | 400 Bad Request | `TestRecordingController_Play_Validation_ZeroDuration` |
| TC-REC-018 | P2 | 認證與卡車權限通過；durationSeconds < 0。 | 執行 `Play`。 | 400 Bad Request | `TestRecordingController_Play_Validation_NegativeDuration` |
| TC-REC-019 | P2 | 認證與卡車權限通過；durationSeconds > 86400 秒。 | 執行 `Play`。 | 400 Bad Request | `TestRecordingController_Play_Validation_TooLongDuration` |
| TC-REC-020 | P2 | 認證與權限通過；合法 start 與 durationSeconds=30；攝影機不存在。 | 執行 `Play`。 | 404 Not Found | `TestRecordingController_Play_Validation_CameraNotFound` |
| TC-REC-021 | P2 | 認證與權限通過；合法 start 與 durationSeconds=30；攝影機查詢回傳 SQL 錯誤。 | 執行 `Play`。 | 500 Internal Server Error | `TestRecordingController_Play_Validation_CameraLookupError` |
| TC-REC-022 | P2 | 認證與權限通過；Body 為 65 KiB 的 x 字元。 | 執行 `Play`。 | 400 Bad Request。此 Body 同時不是合法 JSON，不能單獨證明 64 KiB 上限的邊界行為。 | `TestRecordingController_Play_BodyTooLarge` |
| TC-REC-023 | P2 | 認證、卡車權限及攝影機檢查通過；start=2026-10-20T08:00:00Z、durationSeconds=30.5；記錄執行前後時間。 | 執行 `Play`。 | 200；錄影 content URL 與查詢參數正確；Recording Token 的卡車、攝影機、start、duration 正確；expiresAt 落在 RecordingAccessExpiry 計算的前後界限內。 | `TestRecordingController_Play_Success` |
| TC-REC-024 | P1 | 合法 start 與 duration；未提供錄影 Token。 | 執行 `Content`。 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized_MissingToken` |
| TC-REC-025 | P1 | Token=invalid-token；start 與 duration 合法。 | 執行 `Content`。 | 401 Unauthorized。此字串同時不符合 Token 格式，未隔離「格式正確但簽章遭竄改」的情況。 | `TestRecordingController_Content_Unauthorized_InvalidSignature` |
| TC-REC-026 | P1 | 有效 Recording Token 綁定 other-truck，其餘錄影資訊合法。 | 執行 `Content`。 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized_WrongTruck` |
| TC-REC-027 | P1 | 有效 Recording Token 綁定 other-camera，其餘錄影資訊合法。 | 執行 `Content`。 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized_WrongCamera` |
| TC-REC-028 | P1 | Token 授權視窗為 recordingStart 前 1 小時開始的 30 秒。 | 請求 recordingStart 開始的 30 秒 `Content`。 | 401 Unauthorized；拒絕超出授權錄影視窗的請求。 | `TestRecordingController_Content_Unauthorized_WrongRecordingWindow` |
| TC-REC-029 | P1 | 卡車、攝影機及錄影視窗正確，但 Recording Token 已於 1 分鐘前過期。 | 執行 `Content`。 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized_ExpiredToken` |
| TC-REC-030 | P1 | 有效 Recording Token；缺少 start。 | 執行 `Content`。 | 400 Bad Request | `TestRecordingController_Content_Validation_MissingStart` |
| TC-REC-031 | P1 | 有效 Recording Token；start 時間格式錯誤。 | 執行 `Content`。 | 400 Bad Request | `TestRecordingController_Content_Validation_InvalidStart` |
| TC-REC-032 | P1 | 有效 Recording Token；缺少 duration。 | 執行 `Content`。 | 400 Bad Request | `TestRecordingController_Content_Validation_MissingDuration` |
| TC-REC-033 | P1 | 有效 Recording Token；duration 為非數字。 | 執行 `Content`。 | 400 Bad Request | `TestRecordingController_Content_Validation_InvalidDuration` |
| TC-REC-034 | P1 | 有效 Recording Token；duration = 0。 | 執行 `Content`。 | 400 Bad Request | `TestRecordingController_Content_Validation_ZeroDuration` |
| TC-REC-035 | P1 | 有效 Recording Token；duration < 0。 | 執行 `Content`。 | 400 Bad Request | `TestRecordingController_Content_Validation_NegativeDuration` |
| TC-REC-036 | P1 | 有效 Recording Token；duration > 86400 秒。 | 執行 `Content`。 | 400 Bad Request | `TestRecordingController_Content_Validation_TooLongDuration` |
| TC-REC-037 | P2 | 有效 Recording Token；錄影服務指向拒絕連線的 http://127.0.0.1:1。 | 執行 `Content`。 | 502 Bad Gateway | `TestRecordingController_Content_UpstreamErrors_ConnectionFailed` |
| TC-REC-038 | P2 | 有效 Recording Token；本機 MediaMTX /get 回傳 404。 | 執行 `Content`。 | 502 Bad Gateway | `TestRecordingController_Content_UpstreamErrors_MediaMTXRejected` |
| TC-REC-039 | P2 | 有效 Recording Token；本機 /get 回傳 200、4 bytes 的 ABCD；用戶端沒有 Range。 | 執行 `Content`。 | 200；上游查詢參數、Bearer header 正確，沒有 Range；Body=ABCD；Content-Type=video/mp4、Accept-Ranges=bytes、Content-Length=4、Cache-Control=private, no-store。 | `TestRecordingController_Content_Success_FullFile200` |
| TC-REC-040 | P2 | 有效 Recording Token；用戶端 Range=bytes=0-3；本機 /get fixture 回傳 206 與 4 bytes 的 ABCD。 | 執行 `Content`，下載內容後由 `http.ServeContent` 處理 Range。 | 用戶端得到 206；上游不收到 Range；Body=ABCD、Content-Length=4、Content-Range=bytes 0-3/4；Content-Type、Accept-Ranges、Cache-Control 正確，不直接轉送上游的 bytes 0-3/100。 | `TestRecordingController_Content_Success_PartialFile206` |
| TC-REC-041 | P1 | 無效 Token；合法 start 與 duration。 | 執行 `Content`，檢查錯誤回應。 | 401 Unauthorized；Body 為合法 JSON 且 Content-Type 包含 application/json | `TestRecordingController_Content_ErrorResponseJSON` |

## 6. LocationController

正式路由：`GET /api/trucks/{truckID}/location`、`POST /api/trucks/{truckID}/locations`、`GET /api/trucks/{truckID}/locations/ws`。

測試使用 `truck-1` 與固定時鐘 `2026-10-26T12:00:00Z`，避免停車秒數與寫入時間受實際執行時間影響。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-LOC-001 | P1 | 資料庫為 nil。 | 呼叫 `NewLocationController`。 | Controller、時鐘函式及 subscribers map 均非 nil | `TestNewLocationController` |
| TC-LOC-002 | P1 | 查詢 truck-1，請求沒有已驗證使用者 Context。 | 執行 `Current`。 | 401 Unauthorized；CodeAuthenticationRequired | `TestLocationControllerCurrentRequiresAuthentication` |
| TC-LOC-003 | P2 | 認證與權限通過；回傳 truck-1 定位，stoppedSince 為固定時鐘前 90 秒。 | 執行 `Current`。 | 200；truckId=truck-1；stoppedSeconds=90。 | `TestLocationControllerCurrent_ReturnsCurrentLocation` |
| TC-LOC-004 | P2 | 認證與權限通過；目前定位查詢回傳 sql.ErrNoRows。 | 執行 `Current`。 | 404 Not Found；CodeLocationNotFound | `TestLocationControllerCurrent_ReturnsNotFound` |
| TC-LOC-005 | P2 | 認證與權限通過；目前定位查詢回傳 SQL 錯誤。 | 執行 `Current`。 | 503 Service Unavailable；CodeLocationUnavailable | `TestLocationControllerCurrent_ReportsDatabaseFailure` |
| TC-LOC-006 | P2 | 認證成功；truck-1 權限查詢回傳 false。 | 執行 `Current`。 | 403 Forbidden；CodeTruckAccessDenied | `TestLocationControllerCurrentRejectsUnassignedTruck` |
| TC-LOC-007 | P2 | 認證與權限通過；Body 的 latitude JSON 不完整。 | 執行 `Create`。 | 400 Bad Request；CodeInvalidLocation | `TestLocationControllerCreateRejectsInvalidInput_MalformedJSON` |
| TC-LOC-008 | P2 | 認證與權限通過；定位 JSON 含未知欄位 extra。 | 執行 `Create`。 | 400 Bad Request；CodeInvalidLocation | `TestLocationControllerCreateRejectsInvalidInput_UnknownField` |
| TC-LOC-009 | P2 | 認證與權限通過；latitude=91，其餘必要欄位合法。 | 執行 `Create`。 | 400 Bad Request；CodeInvalidLocation | `TestLocationControllerCreateRejectsInvalidInput_InvalidCoordinates` |
| TC-LOC-010 | P2 | 合法定位輸入；查詢上一筆位置時 SQL 失敗。 | 執行 `Create`。 | 503 Service Unavailable；CodeLocationUnavailable | `TestLocationControllerCreateReportsReadAndWriteFailures_CurrentLocationReadFails` |
| TC-LOC-011 | P2 | 合法定位輸入；上一筆不存在，寫入交易的 Begin 失敗。 | 執行 `Create`。 | 503 Service Unavailable；CodeLocationUnavailable | `TestLocationControllerCreateReportsReadAndWriteFailures_LocationWriteFails` |
| TC-LOC-012 | P2 | 已有停止中的定位與 stop anchor；新定位較新；設定歷史表、狀態表寫入及 Commit 預期，並訂閱 truck-1。 | 執行 `Create`。 | 201；SQL 保留原 stop anchor 與 stoppedSince；回應 truckId、緯度正確；訂閱者收到正確 recordedAt 的廣播。 | `TestLocationControllerCreateSavesAndBroadcastsLocation` |
| TC-LOC-013 | P1 | 建立 truck-1 的訂閱頻道。 | 先廣播其他卡車，再依序廣播 truck-1 的緯度 24.1、24.2。 | 在逾時前從頻道讀到最新緯度 24.2。 | `TestLocationBroadcastKeepsNewestUpdate` |
| TC-LOC-014 | P1 | 同一輛 truck-1 建立兩個訂閱者。 | 依序取消兩個訂閱，記錄每次取消後的狀態。 | 剩一個時數量為 1；全退訂後移除卡車 map 項目 | `TestLocationSubscribeUnsubscribeRemovesEmptyTruckEntry` |
| TC-LOC-015 | P2 | 啟動本機 WebSocket server，掛上認證 middleware，設定預期定位。 | 以 bearer subprotocol 連線、等待訂閱登記，廣播定位、讀取 JSON，再正常關閉。 | 連線、讀取與關閉成功；收到的 truckId 與 latitude 正確。 | `TestLocationControllerStreamSendsBroadcast` |

## 7. SessionController

正式路由：`POST /api/auth/login`、`POST /api/auth/logout`。

密碼目前需為 8–72 bytes，且包含大寫、小寫與數字；Email 目前只接受 `.com` 結尾。登入成功回傳的 API Token 與串流／錄影的簽章 Token 是不同用途。

Login 使用 32 bytes 的安全亂數產生 API Token，再進行 Base64 URL 編碼，Token 有效期限為 24 小時。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-SES-001 | P1 | 資料庫為 nil。 | 呼叫 `NewSessionController`。 | Controller 非 nil；內部 database 維持 nil | `TestNewSessionController` |
| TC-SES-002 | P1 | 合法密碼 Secure123 | 呼叫 `ValidLoginPassword`。 | true | `TestValidLoginPassword_Valid` |
| TC-SES-003 | P1 | password=Short1，不足 8 bytes。 | 呼叫 `ValidLoginPassword`。 | false | `TestValidLoginPassword_TooShort` |
| TC-SES-004 | P1 | 72 個 A 再加 a1，合計 74 bytes。 | 呼叫 `ValidLoginPassword`。 | false | `TestValidLoginPassword_TooLong` |
| TC-SES-005 | P1 | 密碼沒有大寫字母 | 呼叫 `ValidLoginPassword`。 | false | `TestValidLoginPassword_MissingUppercase` |
| TC-SES-006 | P1 | 密碼沒有小寫字母 | 呼叫 `ValidLoginPassword`。 | false | `TestValidLoginPassword_MissingLowercase` |
| TC-SES-007 | P1 | 密碼沒有數字 | 呼叫 `ValidLoginPassword`。 | false | `TestValidLoginPassword_MissingDigit` |
| TC-SES-008 | P1 | 合法 .com Email | 呼叫 `ValidLoginEmail`。 | true | `TestValidLoginEmail_Valid` |
| TC-SES-009 | P1 | email 為前後各有兩個空白的 driver@example.com。 | 呼叫 `ValidLoginEmail`。 | true | `TestValidLoginEmail_TrimsWhitespace` |
| TC-SES-010 | P1 | email=Driver@Example.COM。 | 呼叫 `ValidLoginEmail`。 | true | `TestValidLoginEmail_CaseInsensitive` |
| TC-SES-011 | P1 | email=driver.example.com，缺少 @。 | 呼叫 `ValidLoginEmail`。 | false | `TestValidLoginEmail_MissingAtSign` |
| TC-SES-012 | P1 | email=driver@example.org。 | 呼叫 `ValidLoginEmail`。 | false | `TestValidLoginEmail_UnsupportedDomainSuffix` |
| TC-SES-013 | P1 | email=driver @example.com，含內部空白。 | 呼叫 `ValidLoginEmail`。 | false | `TestValidLoginEmail_EmbeddedWhitespace` |
| TC-SES-014 | P1 | 登入請求 JSON 不完整 | 執行 `Login`。 | 400 Bad Request；CodeInvalidLogin | `TestSessionControllerLoginRejectsInvalidRequest_MalformedJSON` |
| TC-SES-015 | P1 | 登入請求包含未知欄位 extra | 執行 `Login`。 | 400 Bad Request；CodeInvalidLogin | `TestSessionControllerLoginRejectsInvalidRequest_UnknownField` |
| TC-SES-016 | P1 | Email 為目前規則不接受的 .org | 執行 `Login`。 | 400 Bad Request；CodeInvalidLogin | `TestSessionControllerLoginRejectsInvalidRequest_InvalidEmail` |
| TC-SES-017 | P1 | 密碼不符格式規則 | 執行 `Login`。 | 400 Bad Request；CodeInvalidPasswordFormat | `TestSessionControllerLoginRejectsInvalidRequest_InvalidPassword` |
| TC-SES-018 | P2 | 依 Email 查詢使用者時資料庫錯誤 | 執行 `Login`。 | 503 Service Unavailable；CodeAuthenticationUnavailable | `TestSessionControllerLoginHandlesCredentialFailures_DatabaseUnavailable` |
| TC-SES-019 | P2 | 資料庫查無該 Email | 執行 `Login`。 | 401 Unauthorized；CodeInvalidLogin | `TestSessionControllerLoginHandlesCredentialFailures_UserNotFound` |
| TC-SES-020 | P2 | 輸入 Secure123，但資料庫提供 Different123 的 bcrypt hash。 | 執行 `Login`。 | 401 Unauthorized；CodeInvalidLogin | `TestSessionControllerLoginHandlesCredentialFailures_WrongPassword` |
| TC-SES-021 | P2 | 密碼正確但寫入 user_api_tokens SQL 失敗 | 執行 `Login`。 | 503 Service Unavailable；CodeAuthenticationUnavailable | `TestSessionControllerLoginReportsTokenStorageFailure` |
| TC-SES-022 | P2 | Email 前後有空白；帳密正確且 Token 寫入成功；記錄執行前後時間。 | 執行 `Login`。 | 200；以去空白的 Email 查詢；Token 非空；UserID、Email 正確；有效期限落在執行前後各加 24 小時的界限內。 | `TestSessionControllerLoginSuccess` |
| TC-SES-023 | P1 | Authorization header 為空。 | 執行 `Logout`。 | 401 Unauthorized；CodeMissingBearerToken | `TestSessionControllerLogout_MissingBearerToken` |
| TC-SES-024 | P2 | Authorization=Bearer logout-token；刪除該 Token hash 的 SQL 失敗。 | 執行 `Logout`。 | 503 Service Unavailable；CodeLogoutUnavailable | `TestSessionControllerLogout_DatabaseUnavailable` |
| TC-SES-025 | P2 | Authorization=Bearer logout-token；刪除該 Token hash 的 SQL 成功。 | 執行 `Logout`。 | 204 No Content；回應 Body 為空 | `TestSessionControllerLogout_Success` |
| TC-SES-026 | P1 | Authorization: Bearer session-token | 呼叫 `bearerTokenFromRequest`。 | token=session-token；ok=true | `TestBearerTokenFromRequest_Valid` |
| TC-SES-027 | P1 | 小寫 bearer 方案 | 呼叫 `bearerTokenFromRequest`。 | token=session-token；ok=true | `TestBearerTokenFromRequest_CaseInsensitive` |
| TC-SES-028 | P1 | Header 前後及 scheme/token 間有多餘空白 | 呼叫 `bearerTokenFromRequest`。 | token=session-token；ok=true | `TestBearerTokenFromRequest_TrimsWhitespace` |
| TC-SES-029 | P1 | 只有 Bearer 沒有 token | 呼叫 `bearerTokenFromRequest`。 | token 為空；ok=false。 | `TestBearerTokenFromRequest_MissingToken` |
| TC-SES-030 | P1 | Bearer 後只有空白 | 呼叫 `bearerTokenFromRequest`。 | token 為空；ok=false。 | `TestBearerTokenFromRequest_EmptyToken` |
| TC-SES-031 | P1 | Authorization 使用 Basic | 呼叫 `bearerTokenFromRequest`。 | token=session-token；ok=false | `TestBearerTokenFromRequest_WrongScheme` |

## 8. HealthController

正式路由：`GET /health`。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-HLT-001 | P1 | 建立 HealthController、GET /health 請求與 responseRecorder。 | 執行 `Get`。 | 200；Content-Type=application/json；JSON 只有 status 欄位且值為 ok。 | `TestHealthController_Get_Success` |

## 9. TruckController

正式路由：`GET /api/trucks`。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-TRK-001 | P2 | 建立 SQL mock 資料庫連線。 | 呼叫 `NewTruckController`。 | 控制器非 nil，保留傳入的 databaseConnection。 | `TestNewTruckController` |
| TC-TRK-002 | P1 | 資料庫為 nil；請求沒有已驗證使用者 Context。 | 執行 `List`。 | 401；error.code=authentication_required；英文錯誤訊息與 JSON 格式正確。 | `TestTruckController_List_Unauthenticated` |
| TC-TRK-003 | P2 | user001 已驗證；該使用者的卡車查詢回傳 SQL 錯誤。 | 執行 `List`。 | 500；error.code=trucks_unavailable；英文錯誤訊息與 JSON 格式正確。 | `TestTruckController_List_DatabaseError` |
| TC-TRK-004 | P2 | user001 已驗證；回傳 truck001／ABC-1234／online、truck002／XYZ-5678／offline。 | 執行 `List`。 | 200；卡車數量為 2；兩筆的 truckId、plateNo、status 均符合 mock 資料。 | `TestTruckController_List_Success` |
| TC-TRK-005 | P2 | user001 已驗證；卡車查詢沒有資料列。 | 執行 `List`。 | 200；JSON 可解碼為卡車 slice，長度為 0；不強制要求空陣列而非 null。 | `TestTruckController_List_NoAssignedTrucks` |

## 10. StreamController

正式路由：`POST /api/trucks/{truckID}/cameras/{cameraID}/play`。這是即時串流播放，與 RecordingController 的 `/recordings/play` 不同。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-STR-001 | P2 | 建立 SQL mock 資料庫與 streamService。 | 呼叫 `NewStreamController`。 | 控制器非 nil，保留傳入的資料庫與串流服務。 | `TestNewStreamController` |
| TC-STR-002 | P1 | 設定 truck001、cam01，請求沒有已驗證使用者 Context。 | 執行 `Play`。 | 401；error.code=authentication_required；英文錯誤訊息與 JSON 格式正確。 | `TestStreamController_Play_Unauthenticated` |
| TC-STR-003 | P2 | 使用者已驗證；truck001 權限查詢回傳 false。 | 執行 `Play`。 | 403；error.code=truck_access_denied；英文錯誤訊息與 JSON 格式正確。 | `TestStreamController_Play_Forbidden` |
| TC-STR-004 | P2 | 使用者已驗證；truck001 權限查詢回傳 SQL 錯誤。 | 執行 `Play`。 | 503；error.code=authorization_unavailable；英文錯誤訊息與 JSON 格式正確。 | `TestStreamController_Play_AuthorizationDatabaseError` |
| TC-STR-005 | P2 | 認證與卡車權限通過；cam01 存在性查詢回傳 SQL 錯誤。 | 執行 `Play`。 | 500；error.code=camera_lookup_failed；英文錯誤訊息與 JSON 格式正確。 | `TestStreamController_Play_CameraDatabaseError` |
| TC-STR-006 | P2 | 認證與卡車權限通過；cam01 存在性查詢回傳 false。 | 執行 `Play`。 | 404；error.code=camera_not_found；英文錯誤訊息與 JSON 格式正確。 | `TestStreamController_Play_CameraNotFound` |
| TC-STR-007 | P2 | 認證、卡車權限與攝影機檢查通過；設定串流服務，記錄執行前後時間。 | 執行 `Play`。 | 200；truckId、cameraId、main、WebRTC 與 main/whep URL 正確；串流 Token 有效且 claims 正確；期限約 5 分鐘；Token 與回應 expiresAt 在 Unix 秒級相同。 | `TestStreamController_Play_Success` |

## 11. MediaMTXController

正式路由：`POST /internal/mediamtx/auth`。以下案例檢查即時串流 Token 與 Recording Token 的用途隔離。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-MTX-001 | P1 | 以 SignAccess 簽發有效 main 串流 Token；path=truck001/cam01/main、action=read。 | 執行 `Auth`。 | 204 No Content。 | `TestMediaMTXSeparatesLiveAndRecordingTokens_LiveTokenReadsLiveStream` |
| TC-MTX-002 | P1 | 以 SignRecordingAccess 簽發有效 Recording Token；path=truck001/cam01/main、action=playback。 | 執行 `Auth`。 | 204 No Content。 | `TestMediaMTXSeparatesLiveAndRecordingTokens_RecordingTokenPlaysRecording` |
| TC-MTX-003 | P1 | 有效 Recording Token；path=truck001/cam01/main、action=read。 | 執行 `Auth`。 | 401；錄影 Token 不能用於即時串流讀取。 | `TestMediaMTXSeparatesLiveAndRecordingTokens_RecordingTokenCannotReadLiveStream` |
| TC-MTX-004 | P1 | 有效 main 串流 Token；path=truck001/cam01/main、action=playback。 | 執行 `Auth`。 | 401；串流 Token 不能用於錄影播放認證。 | `TestMediaMTXSeparatesLiveAndRecordingTokens_LiveTokenCannotPlayRecording` |

## 12. Authorization

測試對象為共用函式 `currentUser`、`requireTruckAccess`，沒有獨立 HTTP 路由。

已驗證請求由真實 AuthService middleware 配合 SQL mock 建立，使用者為 `user001`／`driver@example.com`／`Test Driver`；同時驗證成功回傳的使用者與失敗時的空使用者。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-AUT-001 | P1 | 建立沒有使用者 Context 的請求。 | 呼叫 `currentUser`。 | 回傳空使用者與 false；401、error.code=authentication_required；英文錯誤訊息與 JSON 格式正確。 | `TestCurrentUser_Unauthenticated` |
| TC-AUT-002 | P2 | 使用真實 middleware 配合 SQL mock 建立 user001 的已驗證請求。 | 呼叫 `currentUser`。 | 回傳完整預期使用者與 true；不寫入 response Body。 | `TestCurrentUser_Authenticated` |
| TC-AUT-003 | P1 | 沒有使用者 Context；requestedTruckID=truck001。 | 呼叫 `requireTruckAccess`。 | 回傳空使用者與 false；401、error.code=authentication_required；英文錯誤訊息與 JSON 格式正確。 | `TestRequireTruckAccess_Unauthenticated` |
| TC-AUT-004 | P2 | user001 已驗證；truck001 權限查詢回傳 SQL 錯誤。 | 呼叫 `requireTruckAccess`。 | 回傳空使用者與 false；503、error.code=authorization_unavailable；英文錯誤訊息與 JSON 格式正確。 | `TestRequireTruckAccess_DatabaseError` |
| TC-AUT-005 | P2 | user001 已驗證；truck001 權限查詢回傳 false。 | 呼叫 `requireTruckAccess`。 | 回傳空使用者與 false；403、error.code=truck_access_denied；英文錯誤訊息與 JSON 格式正確。 | `TestRequireTruckAccess_Forbidden` |
| TC-AUT-006 | P2 | user001 已驗證；truck001 權限查詢回傳 true。 | 呼叫 `requireTruckAccess`。 | 回傳完整預期使用者與 true；不寫入 response Body。 | `TestRequireTruckAccess_Allowed` |

## 13. Response

測試對象為共用函式 `writeJSON`、`writeError`，沒有獨立 HTTP 路由。

| 案例 ID | 階段 | Arrange：前置條件／輸入 | Act：執行操作 | Assert：主要預期結果 | 對應 Test 函式 |
| :--- | :---: | :--- | :--- | :--- | :--- |
| TC-RSP-001 | P1 | expectedStatusCode=201；payload 包含 truckId=truck001、status=online。 | 呼叫 `writeJSON`。 | 201；Content-Type=application/json；解碼後欄位數量及兩個欄位值正確。 | `TestWriteJSON_StatusAndPayload` |
| TC-RSP-002 | P1 | expectedStatusCode=503；error code、message 使用 AuthorizationUnavailable 常數。 | 呼叫 `writeError`。 | 503；Content-Type=application/json；error.code=authorization_unavailable；error.message=Truck permissions could not be verified。 | `TestWriteError_StatusAndErrorBody` |

## 14. 最終測試結果與覆蓋率

### 14.1 執行指令

從專案根目錄執行完整專案測試，並量測 controllers 套件覆蓋率：

```powershell
go test -count=1 '-coverpkg=./controllers' '-coverprofile=controllers-coverage.out' ./...
```

`-count=1` 不使用測試結果快取；`./...` 指定完整專案的測試範圍，`-coverpkg=./controllers` 指定覆蓋率的量測範圍。

### 14.2 最終結果

測試日期：2026-11-10；Go 版本：1.27.1。

| 項目 | 最終結果 |
| :--- | :--- |
| 完整專案測試 | PASS |
| controllers 套件測試 | PASS |
| controllers 套件耗時 | 2.929s |
| controllers 整體 statement coverage | 92.9%（262／282 statements） |
| 未覆蓋 statements | 20 |

耗時為 controllers 套件的執行時間，並非所有 package 的總耗時。

### 14.3 各原始檔覆蓋率

| 原始檔 | 已覆蓋 statements | 總 statements | 覆蓋率 |
| :--- | ---: | ---: | ---: |
| `authorization.go` | 16 | 16 | 100.0% |
| `camera_controller.go` | 23 | 23 | 100.0% |
| `health_controller.go` | 1 | 1 | 100.0% |
| `location_controller.go` | 79 | 83 | 95.2% |
| `mediamtx_controller.go` | 15 | 31 | 48.4% |
| `recording_controller.go` | 61 | 61 | 100.0% |
| `response.go` | 2 | 2 | 100.0% |
| `session_controller.go` | 42 | 42 | 100.0% |
| `stream_controller.go` | 14 | 14 | 100.0% |
| `truck_controller.go` | 9 | 9 | 100.0% |
| **整體** | **262** | **282** | **92.9%** |

整體百分比以已覆蓋 statements 總數除以 statements 總數計算，不是各檔案百分比的算術平均。

### 14.4 尚未覆蓋的部分

| 原始檔 | 未覆蓋 statements | 對應部分 |
| :--- | ---: | :--- |
| `location_controller.go` | 4 | Stream 權限檢查失敗、WebSocket 升級失敗、推播寫入失敗，以及 Create 權限檢查失敗後的 return。 |
| `mediamtx_controller.go` | 16 | 無效 JSON、無效串流 path、publish 流程（憑證檢查與攝影機查詢）以及未知 action 的拒絕處理。 |
| **合計** | **20** | |

以上為 statement coverage 的未覆蓋部分；100% statement coverage 不等同所有條件組合、輸入邊界或端到端情境均已驗證。

## 15. 設計與判讀限制

- 119 個案例代表目前有對應獨立測試函式，不代表所有正式程式分支或輸入邊界均已測試。
- 部分錯誤案例只斷言 HTTP 狀態，未另外驗證錯誤代碼與訊息；不要將正式程式的預期誤列為該案例已有的斷言。
- `TC-REC-022` 的超長輸入同時是非法 JSON，`TC-REC-025` 的 Token 同時不符合格式；案例名稱不能取代實際輸入與斷言範圍。
- MediaMTX 的 4 個案例只涵蓋 read／playback Token 用途，未涵蓋 publish、非法 path 或未知 action 的全部分支。
- 本文件不將真實 PostgreSQL、正式 MediaMTX 或瀏覽器端的端到端行為列為已驗證。
