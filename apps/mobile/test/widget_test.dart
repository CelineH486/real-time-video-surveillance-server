import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/app.dart';
import 'package:surveillance_app/src/screens/login_screen.dart';
import 'package:surveillance_app/src/screens/truck_location_screen.dart';
import 'package:surveillance_app/src/services/session_store.dart';

class MemorySessionStore implements SessionStore {
  String? token;
  String? email;

  @override
  Future<void> deleteToken() async => token = null;

  @override
  Future<String?> readToken() async => token;

  @override
  Future<void> writeToken(String value) async => token = value;

  @override
  Future<String?> readEmail() async => email;

  @override
  Future<void> writeEmail(String value) async => email = value;
}

class HangingSessionStore implements SessionStore {
  final _never = Completer<String?>().future;

  @override
  Future<void> deleteToken() async {}

  @override
  Future<String?> readToken() => _never;

  @override
  Future<void> writeToken(String value) async {}

  @override
  Future<String?> readEmail() => _never;

  @override
  Future<void> writeEmail(String value) async {}
}

void main() {
  testWidgets('未登入時顯示登入頁', (tester) async {
    await tester.pumpWidget(
      SurveillanceApp(sessionStore: MemorySessionStore()),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(find.text('監控系統登入'), findsOneWidget);
    expect(find.text('登入'), findsOneWidget);
  });

  testWidgets('密碼格式不正確時顯示驗證訊息', (tester) async {
    await tester.pumpWidget(
      SurveillanceApp(sessionStore: MemorySessionStore()),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    await tester.enterText(find.byType(EditableText).first, 'dev@example.com');
    await tester.enterText(find.byType(EditableText).last, 'weak');
    await tester.tap(find.text('登入'));
    await tester.pump();

    expect(find.text('至少 8 碼，需包含英文大寫、小寫及數字'), findsOneWidget);
  });

  testWidgets('登入頁可預填保留的電子郵件', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LoginScreen(
          initialEmail: 'dev@example.com',
          onLogin: (_, _) async {},
        ),
      ),
    );

    final emailField = tester.widget<TextFormField>(
      find.byType(TextFormField).first,
    );
    final passwordField = tester.widget<TextFormField>(
      find.byType(TextFormField).last,
    );
    expect(emailField.controller?.text, 'dev@example.com');
    expect(passwordField.controller?.text, isEmpty);
  });

  testWidgets('登入儲存無回應時不會永久停在載入畫面', (tester) async {
    await tester.pumpWidget(
      SurveillanceApp(sessionStore: HangingSessionStore()),
    );

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    await tester.pump(const Duration(seconds: 3));
    await tester.pump();

    expect(find.text('監控系統登入'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });

  testWidgets('已登入時重新整理會恢復網址中的車機頁面', (tester) async {
    final sessionStore = MemorySessionStore()
      ..token = 'test-token'
      ..email = 'dev@example.com';

    await tester.pumpWidget(
      SurveillanceApp(
        sessionStore: sessionStore,
        initialRoute: '/trucks/truck001/location',
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.byType(TruckLocationScreen), findsOneWidget);
    expect(find.text('選擇車機'), findsNothing);
  });
}
