# Сопоставление токенов

Используй существующий формат экспорта проекта, если он уже есть. Если формата нет, создай небольшой manifest рядом с Flutter theme в рамках задачи: canonical Figma path → mode → resolved value → Dart symbol. Не храни вторую вручную поддерживаемую копию палитры внутри skill.

Минимальные поля для воспроизводимого сопоставления:

| Поле | Назначение |
|---|---|
| source file key / collection / variable ID | Однозначный исходный объект, не только display name |
| canonical name | Оригинальный path, например `text/primary` |
| source mode ID + name | Реальный Dark/Light, без угадывания порядка modes |
| type / alias target / resolved value | Тип, цепочка semantic→primitive и итог по каждому mode |
| Dart symbol / code path | Допустимый символ, например `textPrimary`, и место в коде |
| evidence / export date | Чем и когда прочитаны переменные; дата экспорта не дата приёмки UI |

Slash в canonical name не является допустимым Dart-идентификатором. Сохраняй исходное имя в mapping, а правило преобразования делай обратимым и проверяй коллизии. Не разрешай разные `surface/base` и `surfaceBase` в один символ молча.

Сверяй не только hex: тип, modes, alpha, alias/binding, единицы числовых токенов и потребителей. Выявляй отсутствующий mode, alias cycle/missing target, неиспользуемый токен, hardcoded цвет и расхождение resolved value. Не удаляй неиспользуемые токены, пока не проверены их потребители и scope задачи.

Пары контраста выбирай из реальных компонентов: текст/поверхность, label/primary CTA, error/warning, существенные границы. Сохраняй source names, mode, foreground/background, ratio, порог и исход. Для прозрачного текста/слоя учитывай композицию; `contrast.py` намеренно принимает только opaque RGB.

Контрастная пара не подтверждает весь UI: отдельно проверяются 48 dp targets, Semantics, масштаб текста, длинные args и оба режима. Drift между Figma и Flutter оценивай по одному экспорту и конкретной revision кода; при недоступном Figma результат — «не сверено с текущим файлом».
