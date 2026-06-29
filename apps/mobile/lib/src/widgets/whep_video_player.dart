// ignore_for_file: avoid_web_libraries_in_flutter

import 'dart:convert';
import 'dart:html' as html;
import 'dart:ui_web' as ui_web;

import 'package:flutter/material.dart';

class WhepVideoPlayer extends StatefulWidget {
  const WhepVideoPlayer({
    super.key,
    required this.url,
    required this.token,
    this.muted = true,
  });

  final String url;
  final String token;
  final bool muted;

  @override
  State<WhepVideoPlayer> createState() => _WhepVideoPlayerState();
}

class _WhepVideoPlayerState extends State<WhepVideoPlayer> {
  late final String _viewType;
  late final html.IFrameElement _iframe;

  @override
  void initState() {
    super.initState();
    _viewType = 'whep-player-${identityHashCode(this)}';
    _iframe = html.IFrameElement()
      ..style.border = '0'
      ..style.width = '100%'
      ..style.height = '100%'
      ..allow = 'autoplay; fullscreen'
      ..srcdoc = _srcDoc();
    ui_web.platformViewRegistry.registerViewFactory(_viewType, (_) => _iframe);
  }

  @override
  void didUpdateWidget(covariant WhepVideoPlayer oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url != widget.url ||
        oldWidget.token != widget.token ||
        oldWidget.muted != widget.muted) {
      _iframe.srcdoc = _srcDoc();
    }
  }

  @override
  Widget build(BuildContext context) {
    return HtmlElementView(viewType: _viewType);
  }

  String _srcDoc() {
    final url = jsonEncode(widget.url);
    final token = jsonEncode(widget.token);
    final muted = widget.muted ? 'true' : 'false';

    return '''
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    html, body { margin: 0; width: 100%; height: 100%; background: #000; overflow: hidden; }
    video { width: 100%; height: 100%; object-fit: cover; background: #000; }
    .state {
      position: fixed; inset: 0; display: grid; place-items: center;
      color: rgba(255,255,255,.68); font: 13px system-ui, sans-serif;
      background: #000;
    }
    .state.ready { display: none; }
  </style>
</head>
<body>
  <video autoplay playsinline ${widget.muted ? 'muted' : ''}></video>
  <div class="state" id="state">正在連線...</div>
  <script>
    const whepUrl = $url;
    const accessToken = $token;
    const muted = $muted;
    const video = document.querySelector('video');
    const state = document.getElementById('state');
    let pc;
    let sessionUrl;

    async function waitForIceGathering(peer) {
      if (peer.iceGatheringState === 'complete') return;
      await new Promise(resolve => {
        const timeout = setTimeout(resolve, 2500);
        peer.addEventListener('icegatheringstatechange', () => {
          if (peer.iceGatheringState === 'complete') {
            clearTimeout(timeout);
            resolve();
          }
        });
      });
    }

    async function start() {
      try {
        video.muted = muted;
        pc = new RTCPeerConnection();
        pc.addTransceiver('video', { direction: 'recvonly' });
        pc.ontrack = event => {
          video.srcObject = event.streams[0];
          video.play().catch(() => {});
          state.classList.add('ready');
        };
        pc.onconnectionstatechange = () => {
          if (['failed', 'disconnected', 'closed'].includes(pc.connectionState)) {
            state.classList.remove('ready');
            state.textContent = '連線中斷';
          }
        };

        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        await waitForIceGathering(pc);

        const response = await fetch(whepUrl, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer \${accessToken}`,
            'Content-Type': 'application/sdp'
          },
          body: pc.localDescription.sdp
        });
        if (!response.ok) throw new Error(`WHEP \${response.status}`);

        const locationHeader = response.headers.get('Location');
        sessionUrl = locationHeader ? new URL(locationHeader, whepUrl).href : null;
        await pc.setRemoteDescription({ type: 'answer', sdp: await response.text() });
      } catch (error) {
        state.classList.remove('ready');
        state.textContent = `無法播放：\${error.message}`;
      }
    }

    window.addEventListener('pagehide', () => {
      if (sessionUrl) {
        fetch(sessionUrl, { method: 'DELETE', headers: { 'Authorization': `Bearer \${accessToken}` } }).catch(() => {});
      }
      if (pc) pc.close();
    });

    start();
  </script>
</body>
</html>
''';
  }
}
