import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/src/app.dart';

void main() {
  testWidgets('shows truck selection shell', (tester) async {
    await tester.pumpWidget(const SurveillanceApp());

    expect(find.text('選擇車機'), findsOneWidget);
  });
}
