import 'package:flutter/material.dart';

/// Portway semantic colours. Figma file BZ7UxNzlF3OmJRhd5JS19s, collections
/// "Portway — Dark" / "Portway — Light"; mapping in design/figma-tokens.json
/// and DESIGN.md. Roles marked "derived" do not exist in Figma: they are
/// opaque blends or calmer separators introduced by the redesign so that every
/// pair can be contrast-checked (see DESIGN.md § Контраст).
@immutable
class PortwayColors extends ThemeExtension<PortwayColors> {
  const PortwayColors({
    required this.background,
    required this.surface,
    required this.raised,
    required this.field,
    required this.text,
    required this.secondary,
    required this.muted,
    required this.disabled,
    required this.onAccent,
    required this.accent,
    required this.accentContainer,
    required this.warning,
    required this.warningContainer,
    required this.danger,
    required this.dangerContainer,
    required this.controlBorder,
    required this.divider,
  });

  /// background/primary
  final Color background;

  /// Cards, navigation bar. Dark: surface/base; Light: derived white card.
  final Color surface;

  /// surface/raised (dark) / surface/base (light): chips, code, tracks.
  final Color raised;

  /// Text field fill (derived).
  final Color field;

  /// text/primary, text/secondary, text/muted, text/disabled, text/on-accent.
  final Color text, secondary, muted, disabled, onAccent;

  /// accent/primary and its opaque 14 % (dark) / 10 % (light) container.
  final Color accent, accentContainer;

  /// status/warning, status/danger and their opaque containers (derived).
  final Color warning, warningContainer, danger, dangerContainer;

  /// border/control (≥ 3:1 against the surface it sits on).
  final Color controlBorder;

  /// Decorative separators and card outlines. Dark: border/subtle; Light:
  /// derived #dde3e4 because Figma light border/subtle equals border/control.
  final Color divider;

  static const dark = PortwayColors(
    background: Color(0xff0d1112),
    surface: Color(0xff141a1c),
    raised: Color(0xff1d2426),
    field: Color(0xff1d2426),
    text: Color(0xfff3f6f6),
    secondary: Color(0xffadb7ba),
    muted: Color(0xff9ca8ac),
    disabled: Color(0xff778589),
    onAccent: Color(0xff07282b),
    accent: Color(0xff35c6ce),
    accentContainer: Color(0xff193235),
    warning: Color(0xfff2c879),
    warningContainer: Color(0xff333229),
    danger: Color(0xfff09082),
    dangerContainer: Color(0xff332b2a),
    controlBorder: Color(0xff65757a),
    divider: Color(0xff2a3a3d),
  );
  static const light = PortwayColors(
    background: Color(0xfff6f8f8),
    surface: Color(0xffffffff),
    raised: Color(0xffeff2f2),
    field: Color(0xffffffff),
    text: Color(0xff172126),
    secondary: Color(0xff556368),
    muted: Color(0xff5f6d72),
    disabled: Color(0xff839197),
    onAccent: Color(0xfff3f6f6),
    accent: Color(0xff006970),
    accentContainer: Color(0xffe6f0f1),
    warning: Color(0xff7a520b),
    warningContainer: Color(0xfff2eee7),
    danger: Color(0xffa6332f),
    dangerContainer: Color(0xfff6ebea),
    controlBorder: Color(0xff839197),
    divider: Color(0xffdde3e4),
  );

  static PortwayColors of(BuildContext context) =>
      Theme.of(context).extension<PortwayColors>()!;

  @override
  PortwayColors copyWith({
    Color? background,
    Color? surface,
    Color? raised,
    Color? field,
    Color? text,
    Color? secondary,
    Color? muted,
    Color? disabled,
    Color? onAccent,
    Color? accent,
    Color? accentContainer,
    Color? warning,
    Color? warningContainer,
    Color? danger,
    Color? dangerContainer,
    Color? controlBorder,
    Color? divider,
  }) => PortwayColors(
    background: background ?? this.background,
    surface: surface ?? this.surface,
    raised: raised ?? this.raised,
    field: field ?? this.field,
    text: text ?? this.text,
    secondary: secondary ?? this.secondary,
    muted: muted ?? this.muted,
    disabled: disabled ?? this.disabled,
    onAccent: onAccent ?? this.onAccent,
    accent: accent ?? this.accent,
    accentContainer: accentContainer ?? this.accentContainer,
    warning: warning ?? this.warning,
    warningContainer: warningContainer ?? this.warningContainer,
    danger: danger ?? this.danger,
    dangerContainer: dangerContainer ?? this.dangerContainer,
    controlBorder: controlBorder ?? this.controlBorder,
    divider: divider ?? this.divider,
  );

  @override
  PortwayColors lerp(PortwayColors? other, double t) {
    if (other == null) return this;
    Color l(Color a, Color b) => Color.lerp(a, b, t)!;
    return PortwayColors(
      background: l(background, other.background),
      surface: l(surface, other.surface),
      raised: l(raised, other.raised),
      field: l(field, other.field),
      text: l(text, other.text),
      secondary: l(secondary, other.secondary),
      muted: l(muted, other.muted),
      disabled: l(disabled, other.disabled),
      onAccent: l(onAccent, other.onAccent),
      accent: l(accent, other.accent),
      accentContainer: l(accentContainer, other.accentContainer),
      warning: l(warning, other.warning),
      warningContainer: l(warningContainer, other.warningContainer),
      danger: l(danger, other.danger),
      dangerContainer: l(dangerContainer, other.dangerContainer),
      controlBorder: l(controlBorder, other.controlBorder),
      divider: l(divider, other.divider),
    );
  }
}

/// Spacing (4-pt grid) and radii shared by every screen.
abstract final class Space {
  static const xs = 4.0, sm = 8.0, md = 12.0, lg = 16.0, xl = 24.0, xxl = 32.0;
}

abstract final class Radii {
  /// Chips, badges, code blocks.
  static const small = 8.0;

  /// Cards, buttons, fields, banners.
  static const medium = 12.0;

  /// Bottom sheets and dialogs.
  static const large = 24.0;
}
