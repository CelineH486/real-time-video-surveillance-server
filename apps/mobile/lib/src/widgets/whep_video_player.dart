import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:http/http.dart' as http;

class WhepVideoPlayer extends StatefulWidget {
  const WhepVideoPlayer({
    super.key,
    required this.url,
    required this.token,
    this.muted = true,
    this.fit = RTCVideoViewObjectFit.RTCVideoViewObjectFitCover,
    this.onAuthenticationExpired,
  });

  final String url;
  final String token;
  final bool muted;
  final RTCVideoViewObjectFit fit;
  final VoidCallback? onAuthenticationExpired;

  @override
  State<WhepVideoPlayer> createState() => _WhepVideoPlayerState();
}

class _WhepVideoPlayerState extends State<WhepVideoPlayer> {
  static const _reconnectDelay = Duration(seconds: 2);

  final RTCVideoRenderer _renderer = RTCVideoRenderer();
  RTCPeerConnection? _peerConnection;
  Uri? _sessionUri;
  Timer? _reconnectTimer;
  bool _rendererInitialized = false;
  bool _restarting = false;
  bool _disposing = false;
  bool _ready = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    unawaited(_start());
  }

  @override
  void didUpdateWidget(covariant WhepVideoPlayer oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url != widget.url || oldWidget.muted != widget.muted) {
      unawaited(_restart());
    }
  }

  Future<void> _start() async {
    try {
      if (!_rendererInitialized) {
        await _renderer.initialize();
        _rendererInitialized = true;
      }
      _renderer.muted = widget.muted;

      final peerConnection = await createPeerConnection({
        'sdpSemantics': 'unified-plan',
      });
      _peerConnection = peerConnection;
      await peerConnection.addTransceiver(
        kind: RTCRtpMediaType.RTCRtpMediaTypeVideo,
        init: RTCRtpTransceiverInit(direction: TransceiverDirection.RecvOnly),
      );
      peerConnection.onTrack = (event) {
        if (event.streams.isEmpty) return;
        _renderer.srcObject = event.streams.first;
        if (mounted) {
          setState(() {
            _ready = true;
            _error = null;
          });
        }
      };
      peerConnection.onConnectionState = (state) {
        if (!mounted || _disposing) return;
        if (state == RTCPeerConnectionState.RTCPeerConnectionStateFailed ||
            state ==
                RTCPeerConnectionState.RTCPeerConnectionStateDisconnected ||
            state == RTCPeerConnectionState.RTCPeerConnectionStateClosed) {
          setState(() {
            _ready = false;
            _error = '串流連線已中斷，正在重新連線…';
          });
          _scheduleReconnect();
        }
      };

      final offer = await peerConnection.createOffer();
      await peerConnection.setLocalDescription(offer);
      await _waitForIceGathering(peerConnection);
      final localDescription = await peerConnection.getLocalDescription();
      final response = await http.post(
        Uri.parse(widget.url),
        headers: {
          'Authorization': 'Bearer ${widget.token}',
          'Content-Type': 'application/sdp',
        },
        body: localDescription?.sdp ?? offer.sdp,
      );
      if (response.statusCode < 200 || response.statusCode >= 300) {
        if (response.statusCode == 401 || response.statusCode == 403) {
          widget.onAuthenticationExpired?.call();
        }
        throw StateError('WHEP ${response.statusCode}');
      }

      final location = response.headers['location'];
      if (location != null && location.isNotEmpty) {
        _sessionUri = Uri.parse(widget.url).resolve(location);
      }
      await peerConnection.setRemoteDescription(
        RTCSessionDescription(response.body, 'answer'),
      );
    } catch (error) {
      if (mounted) {
        setState(() {
          _ready = false;
          _error = '載入影像失敗：$error';
        });
        _scheduleReconnect();
      }
    }
  }

  void _scheduleReconnect() {
    if (_disposing || _restarting || _reconnectTimer?.isActive == true) return;
    _reconnectTimer = Timer(_reconnectDelay, () => unawaited(_restart()));
  }

  Future<void> _waitForIceGathering(RTCPeerConnection peerConnection) async {
    if (peerConnection.iceGatheringState ==
        RTCIceGatheringState.RTCIceGatheringStateComplete) {
      return;
    }

    final completer = Completer<void>();
    Timer? timeout;
    peerConnection.onIceGatheringState = (state) {
      if (state == RTCIceGatheringState.RTCIceGatheringStateComplete &&
          !completer.isCompleted) {
        timeout?.cancel();
        completer.complete();
      }
    };
    timeout = Timer(const Duration(milliseconds: 2500), () {
      if (!completer.isCompleted) completer.complete();
    });
    await completer.future;
  }

  Future<void> _restart() async {
    if (_disposing || _restarting) return;
    _restarting = true;
    _reconnectTimer?.cancel();
    try {
      await _stop(disposeRenderer: false);
      if (mounted && !_disposing) {
        setState(() {
          _ready = false;
          _error = null;
        });
        await _start();
      }
    } finally {
      _restarting = false;
      if (!_disposing && _error != null) _scheduleReconnect();
    }
  }

  Future<void> _stop({required bool disposeRenderer}) async {
    final sessionUri = _sessionUri;
    _sessionUri = null;
    if (sessionUri != null) {
      try {
        await http.delete(
          sessionUri,
          headers: {'Authorization': 'Bearer ${widget.token}'},
        );
      } catch (_) {
        // The server will expire abandoned WHEP sessions.
      }
    }
    await _peerConnection?.close();
    _peerConnection = null;
    _renderer.srcObject = null;
    if (disposeRenderer) {
      await _renderer.dispose();
      _rendererInitialized = false;
    }
  }

  @override
  void dispose() {
    _disposing = true;
    _reconnectTimer?.cancel();
    unawaited(_stop(disposeRenderer: true));
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: Colors.black,
      child: Stack(
        fit: StackFit.expand,
        children: [
          RTCVideoView(_renderer, objectFit: widget.fit),
          if (!_ready)
            Center(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  _error ?? '正在取得畫面…',
                  textAlign: TextAlign.center,
                  style: const TextStyle(color: Colors.white54),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
