import 'package:flutter/material.dart';

import 'colors.dart';

export 'colors.dart';

/// Monospace for codes, IDs, paths and code blocks (Android system mono).
const monoFamily = 'monospace';

ThemeData portwayTheme(Brightness brightness) {
  final dark = brightness == Brightness.dark;
  final c = dark ? PortwayColors.dark : PortwayColors.light;
  final scheme =
      ColorScheme.fromSeed(
        seedColor: c.accent,
        brightness: brightness,
      ).copyWith(
        primary: c.accent,
        onPrimary: c.onAccent,
        primaryContainer: c.accentContainer,
        onPrimaryContainer: c.text,
        secondary: c.accent,
        onSecondary: c.onAccent,
        secondaryContainer: c.accentContainer,
        onSecondaryContainer: c.accent,
        tertiary: c.warning,
        tertiaryContainer: c.warningContainer,
        onTertiaryContainer: c.text,
        surface: c.background,
        onSurface: c.text,
        onSurfaceVariant: c.muted,
        surfaceContainerLowest: c.background,
        surfaceContainerLow: c.surface,
        surfaceContainer: c.surface,
        surfaceContainerHigh: c.raised,
        surfaceContainerHighest: c.raised,
        surfaceTint: Colors.transparent,
        outline: c.controlBorder,
        outlineVariant: c.divider,
        error: c.danger,
        onError: c.background,
        errorContainer: c.dangerContainer,
        onErrorContainer: c.text,
        inverseSurface: c.text,
        onInverseSurface: c.background,
        inversePrimary: dark
            ? PortwayColors.light.accent
            : PortwayColors.dark.accent,
      );
  final base = ThemeData(
    useMaterial3: true,
    colorScheme: scheme,
    brightness: brightness,
  );
  final t = base.textTheme;
  TextStyle? s(TextStyle? v, double size, FontWeight w, [double h = 1.3]) =>
      v?.copyWith(
        fontSize: size,
        fontWeight: w,
        height: h,
        letterSpacing: 0,
        color: c.text,
      );
  final text = t.copyWith(
    displaySmall: s(t.displaySmall, 32, FontWeight.w600, 1.15),
    headlineMedium: s(t.headlineMedium, 28, FontWeight.w600, 1.2),
    headlineSmall: s(t.headlineSmall, 24, FontWeight.w600, 1.25),
    titleLarge: s(t.titleLarge, 20, FontWeight.w600),
    titleMedium: s(t.titleMedium, 16, FontWeight.w600, 1.35),
    titleSmall: s(t.titleSmall, 14, FontWeight.w600, 1.35),
    bodyLarge: s(t.bodyLarge, 16, FontWeight.w400, 1.5),
    bodyMedium: s(t.bodyMedium, 14, FontWeight.w400, 1.45),
    bodySmall: s(
      t.bodySmall,
      13,
      FontWeight.w400,
      1.4,
    )?.copyWith(color: c.muted),
    labelLarge: s(t.labelLarge, 15, FontWeight.w600, 1.2),
    labelMedium: s(t.labelMedium, 13, FontWeight.w500, 1.25),
    labelSmall: s(t.labelSmall, 12, FontWeight.w500, 1.25),
  );
  final medium = BorderRadius.circular(Radii.medium);
  final shape = RoundedRectangleBorder(borderRadius: medium);
  OutlineInputBorder border(Color color, [double width = 1]) =>
      OutlineInputBorder(
        borderRadius: medium,
        borderSide: BorderSide(color: color, width: width),
      );
  const buttonPadding = EdgeInsets.symmetric(horizontal: 20, vertical: 14);
  return base.copyWith(
    extensions: [c],
    scaffoldBackgroundColor: c.background,
    canvasColor: c.background,
    textTheme: text,
    appBarTheme: AppBarTheme(
      centerTitle: false,
      toolbarHeight: 64,
      elevation: 0,
      scrolledUnderElevation: 0,
      backgroundColor: c.background,
      foregroundColor: c.text,
      surfaceTintColor: Colors.transparent,
      iconTheme: IconThemeData(size: 24, color: c.text),
      actionsIconTheme: IconThemeData(size: 22, color: c.secondary),
      titleSpacing: 16,
      titleTextStyle: text.titleLarge,
    ),
    cardTheme: CardThemeData(
      color: c.surface,
      elevation: 0,
      margin: EdgeInsets.zero,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: medium,
        side: BorderSide(color: c.divider),
      ),
      clipBehavior: Clip.antiAlias,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: c.field,
      floatingLabelBehavior: FloatingLabelBehavior.auto,
      labelStyle: TextStyle(color: c.muted, fontSize: 16),
      floatingLabelStyle: WidgetStateTextStyle.resolveWith(
        (states) => TextStyle(
          color: states.contains(WidgetState.error)
              ? c.danger
              : states.contains(WidgetState.focused)
              ? c.accent
              : c.muted,
        ),
      ),
      hintStyle: TextStyle(color: c.muted),
      helperStyle: TextStyle(color: c.muted, fontSize: 13, height: 1.35),
      errorStyle: TextStyle(color: c.danger, fontSize: 13, height: 1.35),
      helperMaxLines: 3,
      errorMaxLines: 3,
      prefixIconColor: c.muted,
      suffixIconColor: c.muted,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      border: border(c.controlBorder),
      enabledBorder: border(c.controlBorder),
      disabledBorder: border(c.divider),
      focusedBorder: border(c.accent, 2),
      errorBorder: border(c.danger),
      focusedErrorBorder: border(c.danger, 2),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        minimumSize: const Size(64, 52),
        padding: buttonPadding,
        textStyle: text.labelLarge,
        shape: shape,
        disabledBackgroundColor: c.raised,
        disabledForegroundColor: c.disabled,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: c.text,
        minimumSize: const Size(64, 52),
        padding: buttonPadding,
        textStyle: text.labelLarge,
        side: BorderSide(color: c.controlBorder),
        shape: shape,
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: c.accent,
        minimumSize: const Size(48, 48),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
        textStyle: text.labelLarge,
        shape: shape,
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: IconButton.styleFrom(
        minimumSize: const Size(48, 48),
        iconSize: 22,
      ),
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: SegmentedButton.styleFrom(
        minimumSize: const Size(48, 48),
        textStyle: text.labelMedium,
        side: BorderSide(color: c.controlBorder),
        selectedBackgroundColor: c.accentContainer,
        selectedForegroundColor: c.text,
        foregroundColor: c.secondary,
        shape: shape,
      ),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: c.surface,
      selectedColor: c.accentContainer,
      side: BorderSide(color: c.divider),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(Radii.small),
      ),
      labelStyle: text.labelMedium?.copyWith(color: c.text),
      secondaryLabelStyle: text.labelMedium?.copyWith(color: c.text),
      checkmarkColor: c.accent,
      iconTheme: IconThemeData(size: 18, color: c.secondary),
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 6),
      showCheckmark: true,
    ),
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: c.accent,
      foregroundColor: c.onAccent,
      elevation: 2,
      focusElevation: 2,
      hoverElevation: 3,
      highlightElevation: 3,
      extendedTextStyle: text.labelLarge,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: c.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      height: 72,
      indicatorColor: c.accentContainer,
      indicatorShape: const StadiumBorder(),
      labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
      iconTheme: WidgetStateProperty.resolveWith(
        (states) => IconThemeData(
          size: 24,
          color: states.contains(WidgetState.selected) ? c.accent : c.muted,
        ),
      ),
      labelTextStyle: WidgetStateProperty.resolveWith(
        (states) => text.labelMedium?.copyWith(
          color: states.contains(WidgetState.selected) ? c.text : c.muted,
          fontWeight: states.contains(WidgetState.selected)
              ? FontWeight.w600
              : FontWeight.w500,
        ),
      ),
    ),
    navigationRailTheme: NavigationRailThemeData(
      backgroundColor: c.surface,
      indicatorColor: c.accentContainer,
      selectedIconTheme: IconThemeData(size: 24, color: c.accent),
      unselectedIconTheme: IconThemeData(size: 24, color: c.muted),
      selectedLabelTextStyle: text.labelMedium?.copyWith(
        fontWeight: FontWeight.w600,
      ),
      unselectedLabelTextStyle: text.labelMedium?.copyWith(color: c.muted),
    ),
    listTileTheme: ListTileThemeData(
      iconColor: c.secondary,
      textColor: c.text,
      minVerticalPadding: 12,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16),
      titleTextStyle: text.bodyLarge,
      subtitleTextStyle: text.bodySmall,
      shape: shape,
    ),
    dividerTheme: DividerThemeData(color: c.divider, space: 1, thickness: 1),
    progressIndicatorTheme: ProgressIndicatorThemeData(
      color: c.accent,
      linearTrackColor: c.raised,
      circularTrackColor: Colors.transparent,
    ),
    snackBarTheme: SnackBarThemeData(
      behavior: SnackBarBehavior.floating,
      backgroundColor: c.text,
      contentTextStyle: text.bodyMedium?.copyWith(color: c.background),
      actionTextColor: dark
          ? PortwayColors.light.accent
          : PortwayColors.dark.accent,
      shape: shape,
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: c.surface,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(Radii.large),
      ),
      titleTextStyle: text.titleLarge,
      contentTextStyle: text.bodyMedium?.copyWith(color: c.secondary),
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: c.surface,
      surfaceTintColor: Colors.transparent,
      showDragHandle: true,
      dragHandleColor: c.controlBorder,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(Radii.large)),
      ),
    ),
    popupMenuTheme: PopupMenuThemeData(
      color: c.surface,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: medium,
        side: BorderSide(color: c.divider),
      ),
      textStyle: text.bodyMedium,
    ),
    tooltipTheme: TooltipThemeData(
      decoration: BoxDecoration(
        color: c.text,
        borderRadius: BorderRadius.circular(Radii.small),
      ),
      textStyle: text.labelMedium?.copyWith(color: c.background),
    ),
    textSelectionTheme: TextSelectionThemeData(
      cursorColor: c.accent,
      selectionColor: c.accent.withValues(alpha: 0.32),
      selectionHandleColor: c.accent,
    ),
  );
}
