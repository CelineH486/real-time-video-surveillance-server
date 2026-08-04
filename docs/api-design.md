# API / SQL / Flutter 串接草案

這份文件先把目前後端已存在的 API、資料表，以及前後端分離後建議補上的 API 規格整理起來。Web、Flutter、或之後的管理後台都應該只透過 `/api/*` 跟 Go Server 溝通，不直接碰 MediaMTX 或資料庫。

## 目前流程定位

- Truck 端使用 RTSP 推流到 MediaMTX。
- Go Server 負責車機、攝影機、觀看權限、播放 token、歷史影像 API。
- Web / Flutter 先呼叫 Go Server，拿到可觀看的即時串流資訊後，再使用 token 去 MediaMTX 取 WebRTC/WHEP 影像。
- 總覽頁使用每支攝影機的 `sub` 低畫質串流；點開單支攝影機後使用 `main` 高畫質串流。

## 已存在 SQL

### trucks

| 欄位 | 型別 | 說明 |
| --- | --- | --- |
| `truck_id` | `TEXT PRIMARY KEY` | 車機 ID，例如 `truck001` |
| `plate_no` | `TEXT NOT NULL` | 車牌或顯示名稱 |
| `status` | `TEXT NOT NULL DEFAULT 'offline'` | 車機狀態 |

### cameras

| 欄位 | 型別 | 說明 |
| --- | --- | --- |
| `camera_id` | `TEXT PRIMARY KEY` | 攝影機 ID，例如 `cam01` |
| `truck_id` | `TEXT NOT NULL REFERENCES trucks(truck_id)` | 所屬車機 |
| `name` | `TEXT NOT NULL` | 顯示名稱 |
| `status` | `TEXT NOT NULL DEFAULT 'offline'` | 攝影機狀態 |
| `last_seen_at` | `TIMESTAMPTZ` | 最後一次收到影像或狀態時間 |

### recordings

| 欄位 | 型別 | 說明 |
| --- | --- | --- |
| `recording_id` | `BIGSERIAL PRIMARY KEY` | 歷史影像 ID |
| `truck_id` | `TEXT NOT NULL REFERENCES trucks(truck_id)` | 所屬車機 |
| `camera_id` | `TEXT NOT NULL REFERENCES cameras(camera_id)` | 所屬攝影機 |
| `started_at` | `TIMESTAMPTZ NOT NULL` | 開始時間 |
| `ended_at` | `TIMESTAMPTZ NOT NULL` | 結束時間 |
| `file_path` | `TEXT NOT NULL` | 檔案路徑 |
| `file_size` | `BIGINT NOT NULL DEFAULT 0` | 檔案大小 |
| `status` | `TEXT NOT NULL DEFAULT 'ready'` | 錄影狀態 |

## 目前已存在 API

### `GET /api/trucks`

取得車機清單，給 Web / Flutter 做「選車機」。

```json
[
  {
    "truckId": "truck001",
    "plateNo": "TEST-001",
    "status": "offline"
  }
]
```

### `GET /api/trucks/{truckId}/cameras`

取得指定車機的攝影機清單。總覽頁用 `subUrl` + `subToken` 播低畫質串流。

```json
[
  {
    "cameraId": "cam01",
    "truckId": "truck001",
    "name": "Camera 1",
    "status": "online",
    "subUrl": "http://localhost:8889/truck001/cam01/sub/whep",
    "subToken": "signed-token",
    "expiresAt": "2026-06-29T10:00:00Z"
  }
]
```

### `POST /api/trucks/{truckId}/cameras/{cameraId}/play`

取得單支攝影機高畫質即時播放資訊。

```json
{
  "truckId": "truck001",
  "cameraId": "cam01",
  "quality": "main",
  "protocol": "WebRTC",
  "url": "http://localhost:8889/truck001/cam01/main/whep",
  "accessToken": "signed-token",
  "expiresAt": "2026-06-29T10:00:00Z"
}
```

### `GET /api/trucks/{truckId}/recordings?cameraId=cam01`

取得歷史影像列表。

### `POST /api/trucks/{truckId}/cameras/{cameraId}/recordings/play`

取得單段歷史影像播放資料。

## 建議下一階段補上的 API

### 使用者與權限

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `GET /api/users/{userId}/trucks`

原因：未來不會每個使用者都能看全部車機，Flutter / Web 都需要同一套登入與授權機制。

### 車機管理

- `POST /api/trucks`
- `GET /api/trucks/{truckId}`
- `PATCH /api/trucks/{truckId}`
- `DELETE /api/trucks/{truckId}`

### 攝影機管理

- `POST /api/trucks/{truckId}/cameras`
- `PATCH /api/trucks/{truckId}/cameras/{cameraId}`
- `DELETE /api/trucks/{truckId}/cameras/{cameraId}`

### 車機端推流設定

- `GET /api/trucks/{truckId}/stream-config`

回傳 Truck 端需要推流的 RTSP path、帳號，以及心跳設定。密碼建議只在註冊或重設時回傳一次。

```json
{
  "truckId": "truck001",
  "rtspBaseUrl": "rtsp://server.example.com:8554",
  "username": "truck001",
  "cameras": [
    {
      "cameraId": "cam01",
      "mainPath": "/truck001/cam01/main",
      "subPath": "/truck001/cam01/sub"
    }
  ],
  "heartbeat": {
    "protocol": "udp",
    "address": "server.example.com:5000",
    "intervalSeconds": 5
  }
}
```

## Flutter App 從 0 開始的規劃

目前 repo 裡沒有既有 Flutter 專案；Flutter App 會從 0 建立。建議之後建立成獨立前端專案，例如：

```text
apps/
  mobile/
    pubspec.yaml
    lib/
      main.dart
      app.dart
      config/
        api_config.dart
      models/
        truck.dart
        camera.dart
        stream_session.dart
        recording.dart
      services/
        api_client.dart
        truck_api.dart
        camera_api.dart
        stream_api.dart
        recording_api.dart
      screens/
        truck_select_screen.dart
        camera_grid_screen.dart
        camera_live_screen.dart
        recording_list_screen.dart
      widgets/
        camera_tile.dart
        status_badge.dart
```

Flutter App 不需要自己組 RTSP URL，也不應該直接拿永久推流密碼。

建議 Flutter 流程：

1. 登入取得 App API token。
2. 呼叫 `GET /api/trucks` 顯示車機列表。
3. 使用者選車機後，呼叫 `GET /api/trucks/{truckId}/cameras` 顯示總覽。
4. 點單支攝影機時，呼叫 `POST /api/trucks/{truckId}/cameras/{cameraId}/play`。
5. Flutter 播放端使用回傳的 `url` + `accessToken` 播 WebRTC/WHEP。

如果 Flutter 套件不支援 WHEP，可以改走 HLS 或 LL-HLS，但延遲會比 WebRTC 高。即時監控優先建議 WebRTC。

### 初版 Flutter 頁面

初版可以先做 3 個頁面：

1. `TruckSelectScreen`：選車機。
2. `CameraGridScreen`：顯示該 truck 的 8/9 宮格攝影機總覽。
3. `CameraLiveScreen`：點開單支攝影機後播放高畫質即時畫面，並可切到歷史回放。

登入與權限可以先留 API 位置，等車機與播放流程穩定後再補。

### 需要先安裝的本機工具

目前這台環境沒有偵測到 `flutter` 或 `dart` 指令。要真的建立 Flutter 專案，需要先安裝 Flutter SDK，安裝完成後確認：

```powershell
flutter --version
dart --version
flutter doctor
```

確認成功後，建議用以下方式建立：

```powershell
mkdir apps
cd apps
flutter create mobile
```

之後再把 API client、models、screens 接上目前 Go Server 的 `/api/*`。

目前第一版 Flutter Web App 已建立在 `apps/mobile`，預設 API 指向：

```text
http://localhost:8080
```

如果之後要改 API 位置，可以在執行 Flutter 時帶：

```powershell
flutter run -d chrome --dart-define=API_BASE_URL=http://localhost:8080
```

Flutter Web 使用前端路由保存頁面狀態：

- `/trucks`：選車機
- `/trucks/{truckId}/cameras`：指定車機攝影機總覽
- `/trucks/{truckId}/cameras/{cameraId}`：指定攝影機即時頁

因此重新整理頁面時，不會一律回到選車機頁。

## 前後端分離原則

- Go Server 只提供 `/api/*` JSON，不把業務邏輯寫死在 HTML。
- Web 專案未來可以搬到獨立 repo，用環境變數設定 `API_BASE_URL`。
- Flutter、Web 共用同一份 API 文件與 response 格式。
- MediaMTX 的內部 API、RTSP 推流密碼、資料庫連線資訊都不暴露給前端。
- 瀏覽器前端若跟 Go Server 不同網域，需在 Go Server 設定 `CORS_ALLOWED_ORIGINS`，例如 `http://localhost:5173,https://app.example.com`。
