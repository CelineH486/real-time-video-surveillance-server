# Shared Flutter surveillance application

This Flutter/Dart application is the single UI implementation for Web and
mobile-sized clients. It includes:

- email/password login and logout;
- secure session and remembered-email storage;
- responsive truck selection and camera grids;
- WHEP live playback implemented with `flutter_webrtc`;
- historical recording playback implemented with `video_player`.

Run locally against the Go API:

```powershell
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://localhost:8080
```

Build the assets served by the Go application:

```powershell
flutter build web --release --base-href /web/ --no-web-resources-cdn --no-wasm-dry-run
Copy-Item -Path build/web/* -Destination ../../web -Recurse -Force
```

The repository Dockerfile performs this Web build automatically and embeds the
result in the Go binary. Generated files under the repository-level `web/`
directory are ignored; only the Dart source is maintained.
