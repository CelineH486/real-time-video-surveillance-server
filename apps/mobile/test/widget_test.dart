import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/src/app.dart';
import 'package:mobile/src/screens/login_screen.dart';
import 'package:mobile/src/services/session_store.dart';

class MemorySessionStore implements SessionStore {
  String? token;

  @override
  Future<void> deleteToken() async => token = null;

  @override
  Future<String?> readToken() async => token;

  @override
  Future<void> writeToken(String value) async => token = value;
}

void main() {
  testWidgets('未登入時顯示登入頁', (tester) async {
    await tester.pumpWidget(
      SurveillanceApp(sessionStore: MemorySessionStore()),
    );
    await tester.pumpAndSettle();

    expect(find.text('監控系統登入'), findsOneWidget);
    expect(find.text('登入'), findsOneWidget);
  });

  testWidgets('密碼格式不正確時顯示驗證訊息', (tester) async {
    await tester.pumpWidget(
      SurveillanceApp(sessionStore: MemorySessionStore()),
    );
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(EditableText).first, 'dev@example.com');
    await tester.enterText(find.byType(EditableText).last, 'weak');
    await tester.tap(find.text('登入'));
    await tester.pump();

    expect(find.text('至少 8 碼，需包含英文大小寫及數字'), findsOneWidget);
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
}
