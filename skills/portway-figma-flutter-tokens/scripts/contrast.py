#!/usr/bin/env python3
"""Opaque sRGB WCAG contrast; stdout JSON, exit 0=pass, 1=fail, 2=input error.

Formula: https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html
"""

import argparse
import json
import math
import re


def rgb(value):
    """Parse exactly #RGB or #RRGGBB; alpha requires prior composition."""
    if not re.fullmatch(r"#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})", value):
        raise argparse.ArgumentTypeError("Цвет должен быть #RGB или #RRGGBB без alpha")
    digits = value[1:]
    if len(digits) == 3:
        digits = "".join(c * 2 for c in digits)
    return tuple(int(digits[i:i + 2], 16) / 255 for i in (0, 2, 4))


def threshold(value):
    try:
        result = float(value)
    except ValueError as exc:
        raise argparse.ArgumentTypeError("Порог должен быть числом от 1 до 21") from exc
    if not math.isfinite(result) or not 1 <= result <= 21:
        raise argparse.ArgumentTypeError("Порог должен быть конечным числом от 1 до 21")
    return result


def luminance(channels):
    linear = [c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4 for c in channels]
    return sum(c * w for c, w in zip(linear, (0.2126, 0.7152, 0.0722)))


def contrast(foreground, background):
    first, second = sorted((luminance(foreground), luminance(background)))
    return (second + 0.05) / (first + 0.05)


def main():
    parser = argparse.ArgumentParser(description="Контраст непрозрачной sRGB-пары по WCAG")
    parser.add_argument("--foreground", required=True, type=rgb)
    parser.add_argument("--background", required=True, type=rgb)
    parser.add_argument("--threshold", default=4.5, type=threshold)
    args = parser.parse_args()
    ratio = contrast(args.foreground, args.background)
    passed = ratio >= args.threshold
    print(json.dumps({"ratio": ratio, "threshold": args.threshold, "passed": passed}))
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
