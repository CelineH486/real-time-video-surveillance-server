import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/models/camera.dart';
import 'package:surveillance_app/src/models/recording.dart';
import 'package:surveillance_app/src/models/stream_session.dart';
import 'package:surveillance_app/src/screens/camera_live_screen.dart';
import 'package:surveillance_app/src/services/api_client.dart';

class _LayoutApi extends ApiClient {
  @override
  Future<List<Camera>> getCameras(String truckId) async => [
    Camera(
      cameraId: 'cam01',
      truckId: truckId,
      name: '前方道路攝影機',
      status: 'online',
    ),
  ];

  @override
  Future<StreamSession> createPlaySession({
    required String truckId,
    required String cameraId,
  }) async => StreamSession(
    truckId: truckId,
    cameraId: cameraId,
    quality: 'main',
    protocol: 'whep',
    url: 'https://example.invalid/whep',
    accessToken: '',
    expiresAt: DateTime(2027),
  );

  @override
  Future<List<Recording>> getRecordings({
    required String truckId,
    required String cameraId,
    DateTime? start,
    DateTime? end,
  }) async => [];
}

void main() {
  testWidgets('camera mode labels fit on a narrow screen with larger text', (
    tester,
  ) async {
    const channel = MethodChannel('FlutterWebRTC.Method');
    final messenger = tester.binding.defaultBinaryMessenger;
    messenger.setMockMethodCallHandler(channel, (call) async {
      if (call.method == 'createVideoRenderer') return {'textureId': 1};
      if (call.method == 'createPeerConnection') {
        throw PlatformException(code: 'preview-only');
      }
      return null;
    });
    addTearDown(() => messenger.setMockMethodCallHandler(channel, null));
    tester.view.physicalSize = const Size(320, 680);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      MaterialApp(
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: const TextScaler.linear(1.5)),
          child: child!,
        ),
        home: CameraLiveScreen(
          apiClient: _LayoutApi(),
          truckId: 'truck001',
          cameraId: 'cam01',
          onLogout: () async {},
        ),
      ),
    );
    await tester.pump();
    await tester.tap(find.text('歷史錄影'));
    await tester.pump();
    await tester.pump();
    for (final label in ['即時影像', '歷史錄影']) {
      final text = tester.renderObject<RenderParagraph>(find.text(label));
      expect(text.didExceedMaxLines, isFalse);
    }
    expect(find.text('選擇錄影時間'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
