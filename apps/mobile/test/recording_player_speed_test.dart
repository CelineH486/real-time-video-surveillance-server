import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:surveillance_app/src/widgets/recording_player.dart';

void main() {
  testWidgets(
    'offers 8x speed and disables download until a recording is selected',
    (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: RecordingPlayer(
                date: DateTime(2026, 11, 10),
                recordings: const [],
                loadUrl: (_) async => 'http://localhost/test.mp4',
              ),
            ),
          ),
        ),
      );
      final download = tester.widget<IconButton>(
        find.ancestor(
          of: find.byIcon(Icons.download),
          matching: find.byType(IconButton),
        ),
      );
      expect(download.onPressed, isNull);
      await tester.ensureVisible(find.byType(DropdownButton<double>));
      await tester.tap(find.byType(DropdownButton<double>));
      await tester.pumpAndSettle();
      expect(find.text('8.0×'), findsOneWidget);
    },
  );
}
