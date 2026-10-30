- **專案名稱**：Real-time Video Surveillance System（即時影像監控系統）
- **整理依據**：以四份目前提供的 `*_controller_test.go` 程式碼為主；舊版 `api.md` 僅作為文件格式參考。
- **涵蓋模組**：CameraController、RecordingController、LocationController、SessionController。
- **Case 統計原則**：每個一般 `Test...` 函式算一個案例；使用 `t.Run()` 的表格驅動測試，**每個子測試各列一筆**。
- **狀態說明**：表中「已撰寫」僅代表測試程式碼存在，**不表示此處已在你的本機環境執行 PASS**；HTTP 結果為測試中設定的預期。
- **Phase 1（P1）**：不需 SQL Mock 資料庫的檢查，例如建構函式、格式驗證、未驗證請求及純記憶體或 Token 邏輯。
- **Phase 2（P2）**：使用 SQL Mock 或本機測試 HTTP／WebSocket 伺服器驗證控制器與外部依賴互動。

| 模組 | 頂層 Test 函式數 | 展開後 Test Case 數 |
| :--- | ---: | ---: |
| CameraController | 7 | 7 |
| RecordingController | 15 | 41 |
| LocationController | 10 | 15 |
| SessionController | 9 | 31 |
| **合計** | **41** | **94** |

---

## 2. Camera Controller 測試案例（`camera_controller_test.go`）

| 測試案例 ID | 目標功能 | 測試情境 (Scenario) | 階段 | 預期 HTTP 狀態／結果 | 對應測試函式／子測試 |
| :--- | :--- | :--- | :---: | :--- | :--- |
| **TC-CAM-001** | `建構函式` | 以 nil 依賴建立 CameraController | P1 | Controller 實例不為 nil | `TestNewCameraController` |
| **TC-CAM-002** | `List` | 不經驗證 Middleware，沒有已登入使用者 Context | P1 | 401 Unauthorized | `TestCameraController_List_Unauthorized` |
| **TC-CAM-003** | `ListByTruck` | 未驗證使用者查詢指定卡車的攝影機 | P1 | 401 Unauthorized | `TestCameraController_ListByTruck_Unauthorized` |
| **TC-CAM-004** | `List` | Mock Token 驗證成功，回傳可查看的攝影機 cam01 | P2 | 200 OK；回應包含 cam01，SQL 預期均被執行 | `TestCameraController_List_Success` |
| **TC-CAM-005** | `ListByTruck` | 卡車權限通過、查詢 cam01，簽發 sub 串流資訊 | P2 | 200 OK；驗證 subUrl、subToken、claims 與約 5 分鐘有效期限 | `TestCameraController_ListByTruck_Success` |
| **TC-CAM-006** | `List` | 使用者已驗證，但查詢可觀看攝影機的 SQL 回傳錯誤 | P2 | 500 Internal Server Error | `TestCameraController_List_DatabaseError` |
| **TC-CAM-007** | `ListByTruck` | 使用者有卡車權限，但該卡車攝影機查詢發生 SQL 錯誤 | P2 | 500 Internal Server Error | `TestCameraController_ListByTruck_DatabaseError` |


## 3. Recording Controller 測試案例（`recording_controller_test(3).go`）

| 測試案例 ID | 目標功能 | 測試情境 (Scenario) | 階段 | 預期 HTTP 狀態／結果 | 對應測試函式／子測試 |
| :--- | :--- | :--- | :---: | :--- | :--- |
| **TC-REC-001** | `建構函式` | nil 依賴建立控制器 | P1 | Controller 實例不為 nil | `TestNewRecordingController` |
| **TC-REC-002** | `List` | 未設定已登入使用者 Context，直接呼叫 List | P1 | 401 Unauthorized | `TestRecordingController_List_Unauthorized` |
| **TC-REC-003** | `Play` | 未設定已登入使用者 Context，直接呼叫 Play | P1 | 401 Unauthorized | `TestRecordingController_Play_Unauthorized` |
| **TC-REC-004** | `List` | 使用者已登入但沒有指定卡車存取權限 | P2 | 403 Forbidden | `TestRecordingController_List_AccessErrors` / `Forbidden` |
| **TC-REC-005** | `List` | 卡車權限查詢發生 SQL 錯誤 | P2 | 503 Service Unavailable | `TestRecordingController_List_AccessErrors` / `AccessDatabaseError` |
| **TC-REC-006** | `List` | 查詢參數缺少 cameraId | P2 | 400 Bad Request | `TestRecordingController_List_Validation` / `MissingCameraID` |
| **TC-REC-007** | `List` | cameraId 存在，但資料庫找不到攝影機 | P2 | 404 Not Found | `TestRecordingController_List_Validation` / `CameraNotFound` |
| **TC-REC-008** | `List` | 驗證攝影機存在性時 SQL 發生錯誤 | P2 | 500 Internal Server Error | `TestRecordingController_List_Validation` / `CameraLookupError` |
| **TC-REC-009** | `List` | start 不符合 RFC3339 格式 | P2 | 400 Bad Request | `TestRecordingController_List_Validation` / `InvalidStart` |
| **TC-REC-010** | `List` | end 不符合 RFC3339 格式 | P2 | 400 Bad Request | `TestRecordingController_List_Validation` / `InvalidEnd` |
| **TC-REC-011** | `List` | 合法請求經身份/卡車/攝影機驗證，MediaMTX /list 回傳錄影時段 | P2 | 200 OK；確認 MediaMTX 查詢參數及回傳 start、duration=30.5 | `TestRecordingController_List_Success` |
| **TC-REC-012** | `List` | MediaMTX /list 回傳 503 | P2 | 502 Bad Gateway | `TestRecordingController_List_UpstreamError` / `MediaMTXUnavailable` |
| **TC-REC-013** | `List` | MediaMTX /list 回傳 200，但 JSON 格式錯誤 | P2 | 502 Bad Gateway | `TestRecordingController_List_UpstreamError` / `InvalidMediaMTXJSON` |
| **TC-REC-014** | `Play` | 請求本文不是合法 JSON | P2 | 400 Bad Request | `TestRecordingController_Play_Validation` / `InvalidJSON` |
| **TC-REC-015** | `Play` | 缺少 start | P2 | 400 Bad Request | `TestRecordingController_Play_Validation` / `MissingStart` |
| **TC-REC-016** | `Play` | start 格式無效 | P2 | 400 Bad Request | `TestRecordingController_Play_Validation` / `InvalidStart` |
| **TC-REC-017** | `Play` | durationSeconds = 0 | P2 | 400 Bad Request | `TestRecordingController_Play_Validation` / `ZeroDuration` |
| **TC-REC-018** | `Play` | durationSeconds < 0 | P2 | 400 Bad Request | `TestRecordingController_Play_Validation` / `NegativeDuration` |
| **TC-REC-019** | `Play` | durationSeconds > 86400 秒 | P2 | 400 Bad Request | `TestRecordingController_Play_Validation` / `TooLongDuration` |
| **TC-REC-020** | `Play` | 攝影機不存在 | P2 | 404 Not Found | `TestRecordingController_Play_Validation` / `CameraNotFound` |
| **TC-REC-021** | `Play` | 查詢攝影機時 SQL 發生錯誤 | P2 | 500 Internal Server Error | `TestRecordingController_Play_Validation` / `CameraLookupError` |
| **TC-REC-022** | `Play` | 請求 Body 65 KiB，超過 64 KiB 上限 | P2 | 400 Bad Request | `TestRecordingController_Play_BodyTooLarge` |
| **TC-REC-023** | `Play` | 合法播放請求產生 URL 與主碼流 Token，不需真的呼叫 MediaMTX | P2 | 200 OK；驗證 URL、查詢參數、Token claims 與約 5 分鐘有效期限 | `TestRecordingController_Play_Success` |
| **TC-REC-024** | `Content` | 未提供錄影 Token | P1 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized` / `MissingToken` |
| **TC-REC-025** | `Content` | Token 字串無效／簽章不合法 | P1 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized` / `InvalidSignature` |
| **TC-REC-026** | `Content` | Token 對應其他卡車 | P1 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized` / `WrongTruck` |
| **TC-REC-027** | `Content` | Token 對應其他攝影機 | P1 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized` / `WrongCamera` |
| **TC-REC-028** | `Content` | Token 對應 sub 而不是 main | P1 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized` / `WrongQuality` |
| **TC-REC-029** | `Content` | Token 已過期 | P1 | 401 Unauthorized | `TestRecordingController_Content_Unauthorized` / `ExpiredToken` |
| **TC-REC-030** | `Content` | 缺少 start | P1 | 400 Bad Request | `TestRecordingController_Content_Validation` / `MissingStart` |
| **TC-REC-031** | `Content` | start 時間格式錯誤 | P1 | 400 Bad Request | `TestRecordingController_Content_Validation` / `InvalidStart` |
| **TC-REC-032** | `Content` | 缺少 duration | P1 | 400 Bad Request | `TestRecordingController_Content_Validation` / `MissingDuration` |
| **TC-REC-033** | `Content` | duration 為非數字 | P1 | 400 Bad Request | `TestRecordingController_Content_Validation` / `InvalidDuration` |
| **TC-REC-034** | `Content` | duration = 0 | P1 | 400 Bad Request | `TestRecordingController_Content_Validation` / `ZeroDuration` |
| **TC-REC-035** | `Content` | duration < 0 | P1 | 400 Bad Request | `TestRecordingController_Content_Validation` / `NegativeDuration` |
| **TC-REC-036** | `Content` | duration > 86400 秒 | P1 | 400 Bad Request | `TestRecordingController_Content_Validation` / `TooLongDuration` |
| **TC-REC-037** | `Content` | 錄影服務連線失敗（本機端點拒絕連線） | P2 | 502 Bad Gateway | `TestRecordingController_Content_UpstreamErrors` / `ConnectionFailed` |
| **TC-REC-038** | `Content` | MediaMTX /get 回傳 404 非成功回應 | P2 | 502 Bad Gateway | `TestRecordingController_Content_UpstreamErrors` / `MediaMTXRejected` |
| **TC-REC-039** | `Content` | MediaMTX 回傳完整內容；檢查 /get 參數、標頭、ABCD 資料 | P2 | 200 OK；內容 ABCD，回應標頭正確 | `TestRecordingController_Content_Success` / `FullFile200` |
| **TC-REC-040** | `Content` | Range: bytes=0-3，MediaMTX 回傳部分內容 | P2 | 206 Partial Content；Range、Content-Range、ABCD 正確 | `TestRecordingController_Content_Success` / `PartialFile206` |
| **TC-REC-041** | `Content` | 使用無效 Token 請求 Content，驗證錯誤回應格式 | P1 | 401 Unauthorized；Body 為合法 JSON 且 Content-Type 包含 application/json | `TestRecordingController_Content_ErrorResponseJSON` |


## 4. Location Controller 測試案例（`location_controller_test(1).go`）

| 測試案例 ID | 目標功能 | 測試情境 (Scenario) | 階段 | 預期 HTTP 狀態／結果 | 對應測試函式／子測試 |
| :--- | :--- | :--- | :---: | :--- | :--- |
| **TC-LOC-001** | `建構函式` | 傳入 nil 建立 Controller | P1 | Controller、時鐘函式及 subscribers map 均非 nil | `TestNewLocationController` |
| **TC-LOC-002** | `Current` | 沒有已驗證使用者 Context，直接查詢目前位置 | P1 | 401 Unauthorized；CodeAuthenticationRequired | `TestLocationControllerCurrentRequiresAuthentication` |
| **TC-LOC-003** | `Current` | 回傳 truck-1 定位資料；使用固定時間檢查停車秒數 | P2 | 200 OK；truckID 正確、StoppedSeconds = 90 | `TestLocationControllerCurrent` / `returns current location` |
| **TC-LOC-004** | `Current` | 位置查詢回傳 sql.ErrNoRows | P2 | 404 Not Found；CodeLocationNotFound | `TestLocationControllerCurrent` / `returns not found` |
| **TC-LOC-005** | `Current` | 查詢當前位置時 SQL 發生錯誤 | P2 | 503 Service Unavailable；CodeLocationUnavailable | `TestLocationControllerCurrent` / `reports database failure` |
| **TC-LOC-006** | `Current` | 已登入，但未被指派 truck-1 權限 | P2 | 403 Forbidden；CodeTruckAccessDenied | `TestLocationControllerCurrentRejectsUnassignedTruck` |
| **TC-LOC-007** | `Create` | Create 提交損壞 JSON | P2 | 400 Bad Request；CodeInvalidLocation | `TestLocationControllerCreateRejectsInvalidInput` / `malformed JSON` |
| **TC-LOC-008** | `Create` | Create JSON 包含未知欄位 extra | P2 | 400 Bad Request；CodeInvalidLocation | `TestLocationControllerCreateRejectsInvalidInput` / `unknown field` |
| **TC-LOC-009** | `Create` | Create 緯度 91，超出合法範圍 | P2 | 400 Bad Request；CodeInvalidLocation | `TestLocationControllerCreateRejectsInvalidInput` / `invalid coordinates` |
| **TC-LOC-010** | `Create` | 讀取前一筆位置 SQL 失敗 | P2 | 503 Service Unavailable；CodeLocationUnavailable | `TestLocationControllerCreateReportsReadAndWriteFailures` / `current location read fails` |
| **TC-LOC-011** | `Create` | 前一筆不存在，寫入交易 BEGIN 失敗 | P2 | 503 Service Unavailable；CodeLocationUnavailable | `TestLocationControllerCreateReportsReadAndWriteFailures` / `location write fails` |
| **TC-LOC-012** | `Create` | 位置資料成功寫入歷史表與狀態表，並向訂閱者廣播 | P2 | 201 Created；驗證 SQL、回應座標及推播的 recordedAt | `TestLocationControllerCreateSavesAndBroadcastsLocation` |
| **TC-LOC-013** | `broadcast` | 先推播其他卡車，再連續推送 24.1 和 24.2 到同輛卡車 | P1 | 頻道只取得 truck-1 的最新位置：緯度 24.2 | `TestLocationBroadcastKeepsNewestUpdate` |
| **TC-LOC-014** | `subscribe／unsubscribe` | 同卡車加入兩個訂閱者，逐一退訂 | P1 | 剩一個時數量為 1；全退訂後移除卡車 map 項目 | `TestLocationSubscribeUnsubscribeRemovesEmptyTruckEntry` |
| **TC-LOC-015** | `Stream` | 透過本機 HTTP Server 升級 WebSocket，發送定位廣播並讀取 JSON | P2 | 成功收到 truck-1 的位置資料，連線可正常關閉 | `TestLocationControllerStreamSendsBroadcast` |


## 5. Session Controller 測試案例（`session_controller_test.go`）

| 測試案例 ID | 目標功能 | 測試情境 (Scenario) | 階段 | 預期 HTTP 狀態／結果 | 對應測試函式／子測試 |
| :--- | :--- | :--- | :---: | :--- | :--- |
| **TC-SES-001** | `建構函式` | 使用 nil 資料庫建立控制器 | P1 | Controller 非 nil；內部 database 維持 nil | `TestNewSessionController` |
| **TC-SES-002** | `ValidLoginPassword` | 合法密碼 Secure123 | P1 | true | `TestValidLoginPassword` / `valid` |
| **TC-SES-003** | `ValidLoginPassword` | 密碼不足 8 字元 | P1 | false | `TestValidLoginPassword` / `too short` |
| **TC-SES-004** | `ValidLoginPassword` | 密碼長度超過 72 字元 | P1 | false | `TestValidLoginPassword` / `too long` |
| **TC-SES-005** | `ValidLoginPassword` | 密碼沒有大寫字母 | P1 | false | `TestValidLoginPassword` / `missing uppercase` |
| **TC-SES-006** | `ValidLoginPassword` | 密碼沒有小寫字母 | P1 | false | `TestValidLoginPassword` / `missing lowercase` |
| **TC-SES-007** | `ValidLoginPassword` | 密碼沒有數字 | P1 | false | `TestValidLoginPassword` / `missing digit` |
| **TC-SES-008** | `ValidLoginEmail` | 合法 .com Email | P1 | true | `TestValidLoginEmail` / `valid` |
| **TC-SES-009** | `ValidLoginEmail` | 前後包含空白，但去空白後符合格式 | P1 | true | `TestValidLoginEmail` / `trims whitespace` |
| **TC-SES-010** | `ValidLoginEmail` | Email 有大小寫混用 | P1 | true | `TestValidLoginEmail` / `case insensitive` |
| **TC-SES-011** | `ValidLoginEmail` | 缺少 @ | P1 | false | `TestValidLoginEmail` / `missing at sign` |
| **TC-SES-012** | `ValidLoginEmail` | 使用 .org 而非目前規則允許的 .com | P1 | false | `TestValidLoginEmail` / `unsupported domain suffix` |
| **TC-SES-013** | `ValidLoginEmail` | Email 中間含空白 | P1 | false | `TestValidLoginEmail` / `embedded whitespace` |
| **TC-SES-014** | `Login` | 登入請求 JSON 不完整 | P1 | 400 Bad Request；CodeInvalidLogin | `TestSessionControllerLoginRejectsInvalidRequest` / `malformed JSON` |
| **TC-SES-015** | `Login` | 登入請求包含未知欄位 extra | P1 | 400 Bad Request；CodeInvalidLogin | `TestSessionControllerLoginRejectsInvalidRequest` / `unknown field` |
| **TC-SES-016** | `Login` | Email 為目前規則不接受的 .org | P1 | 400 Bad Request；CodeInvalidLogin | `TestSessionControllerLoginRejectsInvalidRequest` / `invalid email` |
| **TC-SES-017** | `Login` | 密碼不符格式規則 | P1 | 400 Bad Request；CodeInvalidPasswordFormat | `TestSessionControllerLoginRejectsInvalidRequest` / `invalid password` |
| **TC-SES-018** | `Login` | 依 Email 查詢使用者時資料庫錯誤 | P2 | 503 Service Unavailable；CodeAuthenticationUnavailable | `TestSessionControllerLoginHandlesCredentialFailures` / `database unavailable` |
| **TC-SES-019** | `Login` | 資料庫查無該 Email | P2 | 401 Unauthorized；CodeInvalidLogin | `TestSessionControllerLoginHandlesCredentialFailures` / `user not found` |
| **TC-SES-020** | `Login` | 有使用者，但 bcrypt 密碼不符 | P2 | 401 Unauthorized；CodeInvalidLogin | `TestSessionControllerLoginHandlesCredentialFailures` / `wrong password` |
| **TC-SES-021** | `Login` | 密碼正確但寫入 user_api_tokens SQL 失敗 | P2 | 503 Service Unavailable；CodeAuthenticationUnavailable | `TestSessionControllerLoginReportsTokenStorageFailure` |
| **TC-SES-022** | `Login` | 成功登入，Email 前後空白能正規化；成功存入 Token | P2 | 200 OK；Token 非空、使用者正確、有效期限約 24 小時 | `TestSessionControllerLoginSuccess` |
| **TC-SES-023** | `Logout` | 登出請求缺少 Bearer Token | P1 | 401 Unauthorized；CodeMissingBearerToken | `TestSessionControllerLogout` / `missing bearer token` |
| **TC-SES-024** | `Logout` | 刪除 Token 的 SQL 發生錯誤 | P2 | 503 Service Unavailable；CodeLogoutUnavailable | `TestSessionControllerLogout` / `database unavailable` |
| **TC-SES-025** | `Logout` | 刪除 Token 成功 | P2 | 204 No Content；回應 Body 為空 | `TestSessionControllerLogout` / `success` |
| **TC-SES-026** | `bearerTokenFromRequest` | Authorization: Bearer session-token | P1 | token=session-token；ok=true | `TestBearerTokenFromRequest` / `valid` |
| **TC-SES-027** | `bearerTokenFromRequest` | 小寫 bearer 方案 | P1 | token=session-token；ok=true | `TestBearerTokenFromRequest` / `case insensitive` |
| **TC-SES-028** | `bearerTokenFromRequest` | Header 前後及 scheme/token 間有多餘空白 | P1 | token=session-token；ok=true | `TestBearerTokenFromRequest` / `trims whitespace` |
| **TC-SES-029** | `bearerTokenFromRequest` | 只有 Bearer 沒有 token | P1 | ok=false | `TestBearerTokenFromRequest` / `missing token` |
| **TC-SES-030** | `bearerTokenFromRequest` | Bearer 後只有空白 | P1 | ok=false | `TestBearerTokenFromRequest` / `empty token` |
| **TC-SES-031** | `bearerTokenFromRequest` | Authorization 使用 Basic | P1 | token=session-token；ok=false | `TestBearerTokenFromRequest` / `wrong scheme` |


---



## 6. 判讀原則

- 「HTTP 401」代表該 Case **預期**未通過身分驗證；「HTTP 403」代表權限不足；實際狀態須透過執行結果確認。
- `sqlmock` 控制假資料庫的查詢／錯誤結果；`httptest` 建立假的 HTTP 請求、回應記錄器與必要的本機伺服器。
