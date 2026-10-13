# Контраст и draft manifest

Статус: **локальные расчёты выполнены; manifest подготовлен как draft, Figma↔Flutter drift NOT_VERIFIED**. Дата: 13.10.2026. Figma-дизайн существует, живого доступа в испытании нет. Flutter-клиент ещё не реализован; проект/ThemeData не создавались.

## Фактические запуски утилиты

Рабочая директория `/home/mushi/Workspace/Portway`, Python 3, утилита `skills/portway-figma-flutter-tokens/scripts/contrast.py`. Прочитан исходный код: непрозрачные sRGB #RGB/#RRGGBB, относительная яркость, вывод JSON; exit 0=pass, 1=fail, 2=input error. Здесь используются только непрозрачные 6-digit RGB, alpha composition не требуется.

```bash
python3 skills/portway-figma-flutter-tokens/scripts/contrast.py --foreground '#f3f6f6' --background '#0d1112' --threshold 4.5
# exit 0
{"ratio": 17.471549999949602, "threshold": 4.5, "passed": true}

python3 skills/portway-figma-flutter-tokens/scripts/contrast.py --foreground '#f3f6f6' --background '#35c6ce' --threshold 4.5
# exit 1
{"ratio": 1.9094386065226798, "threshold": 4.5, "passed": false}

python3 skills/portway-figma-flutter-tokens/scripts/contrast.py --foreground '#0d1112' --background '#35c6ce' --threshold 4.5
# exit 0
{"ratio": 9.150097803755743, "threshold": 4.5, "passed": true}
```

| Foreground / background | Ratio | Обычный текст ≥4.5 | Крупный/значимая граница ≥3 |
|---|---:|---|---|
| `#f3f6f6` / `#0d1112` | 17.47155:1 | PASS | PASS по той же ratio |
| `#f3f6f6` / `#35c6ce` | 1.90944:1 | FAIL | FAIL по той же ratio |
| `#0d1112` / `#35c6ce` | 9.15010:1 | PASS | PASS по той же ratio |

Три CLI-запуска использовали threshold 4.5; столбец 3:1 — сравнение уже полученных чисел с порогом DOC-061 §8, не дополнительный запуск. Вывод при exit 1 — ожидаемый результат недоступной пары, не ошибка выполнения утилиты.

Рабочий label primary CTA: **«Войти», цвет `#0d1112` на `#35c6ce`**. Предлагаемая связь роли `text/on-accent` с `neutral/950` должна быть проверена/принята в Figma. Не менять label на светлый при переходе к Light автоматически: для неизменной opaque teal-пары тёмный label проходит; реальная Light-палитра, overlays и disabled неизвестны и требуют отдельного измерения. Контраст не доказывает target 48 dp, размер/вес крупного текста, Semantics или доступность экрана.

## Draft manifest: известное и неизвестное

Три primitive name/value взяты **из исходного брифа**, не из Figma variables. Canonical semantic paths ниже существуют как примеры в брифе; их существование и binding в текущем файле не подтверждены. Поэтому actual source IDs, mode, alias, resolved value и code path — null. Предложения отделены полями `proposed_*`; это не применённые mappings. `prepared_on` — дата подготовки, `source_exported_at` остаётся null.

```json
{
  "schema": "portway-token-mapping-draft/1",
  "status": "DRAFT_NOT_A_FIGMA_EXPORT",
  "prepared_on": "2026-10-13",
  "source_exported_at": null,
  "source": {
    "kind": "local_design_brief",
    "path": "/home/mushi/Workspace/Portway/skills/portway-figma-flutter-tokens/references/design-brief.md",
    "figma_file_key": null,
    "collection_id": null,
    "collection_name": null,
    "variables_read": false,
    "live_figma_access": false
  },
  "implementation": {
    "flutter_project_exists": false,
    "code_path": null,
    "theme_data_exported": false,
    "theme_extension_exported": false
  },
  "brief_primitives": [
    {"canonical_name": "neutral/950", "type": "COLOR", "opaque_srgb": "#0d1112", "source_status": "brief_value_not_figma_verified"},
    {"canonical_name": "neutral/50", "type": "COLOR", "opaque_srgb": "#f3f6f6", "source_status": "brief_value_not_figma_verified"},
    {"canonical_name": "teal/400", "type": "COLOR", "opaque_srgb": "#35c6ce", "source_status": "brief_value_not_figma_verified"}
  ],
  "mapping_draft": [
    {
      "canonical_name": "text/on-accent",
      "variable_id": null,
      "source_mode_id": null,
      "source_mode_name": null,
      "type": null,
      "alias_target": null,
      "resolved_value": null,
      "dart_symbol": null,
      "code_path": null,
      "proposed_type": "COLOR",
      "proposed_alias_target": "neutral/950",
      "proposed_opaque_srgb": "#0d1112",
      "proposed_dart_symbol": "textOnAccent",
      "evidence": "local contrast calculation with #35c6ce; current Figma binding unknown"
    },
    {
      "canonical_name": "accent/primary",
      "variable_id": null,
      "source_mode_id": null,
      "source_mode_name": null,
      "type": null,
      "alias_target": null,
      "resolved_value": null,
      "dart_symbol": null,
      "code_path": null,
      "proposed_type": "COLOR",
      "proposed_alias_target": "teal/400",
      "proposed_opaque_srgb": "#35c6ce",
      "proposed_dart_symbol": "accentPrimary",
      "evidence": "brief candidate for primary CTA; current Figma binding unknown"
    }
  ],
  "verified_figma_bindings": [],
  "mode_validation": {
    "Dark": "NOT_VERIFIED",
    "Light": "NOT_VERIFIED",
    "actual_mode_ids": [],
    "missing_mode_claim": false
  },
  "contrast_checks": [
    {"foreground": "#f3f6f6", "background": "#0d1112", "mode": null, "ratio": 17.471549999949602, "threshold": 4.5, "passed": true, "exit_code": 0},
    {"foreground": "#f3f6f6", "background": "#35c6ce", "mode": null, "ratio": 1.9094386065226798, "threshold": 4.5, "passed": false, "exit_code": 1},
    {"foreground": "#0d1112", "background": "#35c6ce", "mode": null, "ratio": 9.150097803755743, "threshold": 4.5, "passed": true, "exit_code": 0}
  ],
  "pending": [
    "identify Figma by file key or link, not display name alone",
    "read actual collections, modes, variables, type, alpha and alias chains",
    "read real component consumers and verify bindings, not just matching hex",
    "map canonical paths to explicit Dart symbols and detect collisions",
    "establish existing Flutter code path when implementation exists",
    "calculate real component contrast separately for Dark and Light",
    "check alias cycles, missing targets, units and export/code revision drift",
    "render, large-font, TalkBack, build/goldens and physical-device QA"
  ]
}
```

Slash path сохраняется в canonical name; Dart symbol не выводится без обратимой mapping-таблицы. Два предлагаемых символа различны, но **глобальные коллизии не проверены**: полный набор переменных недоступен. Например `surface/base` и `surfaceBase` нельзя молча свести в один символ. Modes не угадываются по порядку; одинаковый hex не считается binding.

## Фактически выполнено / ограничения / источники

Выполнены 3 реальных subprocess-запуска предоставленной утилиты; подготовлен draft с known brief values и явно неизвестными Figma/code полями. Source names/modes для самих контрастных пар неизвестны, поэтому `mode: null`; вычисление математической пары не назначает её mode.

Не выполнялись live Figma, alias resolution, current bindings/coverage, Light audit, drift, Flutter build/goldens, render/device QA или приёмка владельцем. DOC-061/DOC-020/status/plan только прочитаны, не обновлены в пределах read-only испытания.

Источники: `/home/mushi/Workspace/Portway/skills/portway-figma-flutter-tokens/SKILL.md`, `references/token-mapping.md`, `references/design-brief.md`, `scripts/contrast.py`; `/home/mushi/Workspace/Portway/docs/61-android-screens-flows-and-api.md` §8. Результат 01 содержит визуальное предложение из Comet JPG. Числа/перечни старых дефектов не выдаются за новый аудит.
