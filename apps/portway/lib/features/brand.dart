import 'package:flutter/material.dart';

import '../core/theme.dart';

/// Portway mark: a rounded accent tile with a "P" whose stem ends in a port
/// dot. Same geometry as the Android launcher/splash vectors
/// (android/app/src/main/res/drawable/ic_launcher_foreground.xml).
class BrandMark extends StatelessWidget {
  const BrandMark({super.key, this.size = 40});
  final double size;
  @override
  Widget build(BuildContext context) {
    final c = PortwayColors.of(context);
    return Semantics(
      label: 'Portway',
      image: true,
      child: SizedBox.square(
        dimension: size,
        child: CustomPaint(
          painter: _MarkPainter(tile: c.accent, glyph: c.onAccent),
        ),
      ),
    );
  }
}

class _MarkPainter extends CustomPainter {
  const _MarkPainter({required this.tile, required this.glyph});
  final Color tile, glyph;
  @override
  void paint(Canvas canvas, Size size) {
    final k = size.width / 100;
    canvas.drawRRect(
      RRect.fromRectAndRadius(Offset.zero & size, Radius.circular(28 * k)),
      Paint()..color = tile,
    );
    final stroke = Paint()
      ..color = glyph
      ..style = PaintingStyle.stroke
      ..strokeWidth = 11 * k
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    final p = Path()
      ..moveTo(34 * k, 74 * k)
      ..lineTo(34 * k, 27 * k)
      ..lineTo(53 * k, 27 * k)
      ..arcToPoint(Offset(53 * k, 57 * k), radius: Radius.circular(15 * k))
      ..lineTo(34 * k, 57 * k);
    canvas.drawPath(p, stroke);
    canvas.drawCircle(Offset(67 * k, 72 * k), 6.5 * k, Paint()..color = glyph);
  }

  @override
  bool shouldRepaint(_MarkPainter old) =>
      old.tile != tile || old.glyph != glyph;
}

/// Mark + wordmark used in auth headers.
class BrandLockup extends StatelessWidget {
  const BrandLockup({super.key, this.size = 36});
  final double size;
  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      BrandMark(size: size),
      const SizedBox(width: Space.md),
      Text(
        'Portway',
        style: Theme.of(context).textTheme.titleLarge
            ?.copyWith(fontWeight: FontWeight.w700),
      ),
    ],
  );
}

/// UI-01: shown while the stored session is restored. The mark sits exactly
/// in the centre like the native Android splash, so the hand-off is seamless.
class SplashView extends StatelessWidget {
  const SplashView({super.key});
  static const markSize = 96.0;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context), c = PortwayColors.of(context);
    return Scaffold(
      body: LayoutBuilder(
        builder: (context, box) => Stack(
          children: [
            const Center(child: BrandMark(size: markSize)),
            Positioned(
              left: Space.xl,
              right: Space.xl,
              top: box.maxHeight / 2 + markSize / 2 + Space.xl,
              child: Column(
                children: [
                  Text(
                    'Portway',
                    style: theme.textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: Space.sm),
                  Text(
                    'ИИ-агенты вашего компьютера — с телефона',
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodyMedium?.copyWith(color: c.muted),
                  ),
                ],
              ),
            ),
            Align(
              alignment: Alignment.bottomCenter,
              child: SafeArea(
                child: Padding(
                  padding: const EdgeInsets.only(bottom: 48),
                  child: SizedBox(
                    width: 120,
                    child: LinearProgressIndicator(
                      minHeight: 3,
                      borderRadius: BorderRadius.circular(2),
                      semanticsLabel: 'Загрузка',
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
