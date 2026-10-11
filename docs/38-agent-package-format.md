# Portway — формат переносимого [пакета][g-package]

<a id="doc-038"></a>

| Поле | Значение |
|---|---|
| Артефакт | [DOC-038][id-doc-038] |
| Версия и дата | v0.1 · 10.10.2026 |
| Статус | Проектный формат `portway.agent-package/1`; [import][g-export-import]/[export][g-export-import] NOT_RUN |
| Основания | [FR-060][id-fr-060]–064/077/088, [D-11][id-d-11]; [SRS defaults](03-software-requirements-specification.md#defaults) |
| Потребители | [API packages](36-api-events-and-messages.md#packages), [модель данных](29-data-and-state-model.md), [Runtime/policy](40-agent-execution-and-security.md), [Android](61-android-screens-flows-and-api.md), [перенос и тесты](57-verification-and-traceability.md) |

<a id="purpose"></a>
## 1. Граница [переносимости][g-portability]

[Пакет][g-package] описывает одну [AgentVersion][g-agent-version]: инструкции, [роль][g-role] и заявленные права, [навыки][g-skill], [инструменты][g-tool]/[MCP][g-mcp], требования к [модели][g-model] и переносимые aliases. Это конфигурация, а не исполняемый installation bundle. Он не включает [provider][g-provider] secret values, master key, DB/[FCM][g-push] credentials, [device session][g-device-session], текущий [run][g-run]/[workspace][g-workspace], полную память [LLM][g-model], исходные host paths и обязательные локальные [UUID][g-identifier-formats]. Перенос подтверждается новой работой на другой [модели][g-model] и отдельном [хосте][g-host], не просто успешной распаковкой.

Единственный источник [экспорта][g-export-import] — выбранная неизменяемая [AgentVersion][g-agent-version]. Локальные bindings заменяются aliases, а требования и portable [policy][g-policy] сохраняются. Если конкретную настройку нельзя представить форматом v1, [экспорт][g-export-import] возвращает явную UNSUPPORTED_PACKAGE_FORMAT с безопасным перечнем несовместимых полей; молчаливое удаление настроек не допускается.

<a id="layout"></a>
## 2. Состав [ZIP][g-package] и файловый контракт

```text
manifest.json
AGENTS.md
mcp.json
skills/
  <skill-slug>/
    SKILL.md
    resources/<relative-resource>   # необязательно, инертные объявленные ресурсы
```

Manifest, AGENTS.md и mcp.json обязательны; каталог [skills][g-skill] может быть пуст. Каждый [skill][g-skill] имеет SKILL.md; дополнительные ресурсы объявляются в manifest, читаются как данные и никогда не исполняются при [импорте][g-export-import]/[activation][g-staging]. [Пакет][g-package] не регистрирует собственные executable [tools][g-tool] или [MCP][g-mcp] launch commands. Рабочий [инструмент][g-tool] сопоставляется с уже установленным allowlisted adapter на цели.

В [ZIP][g-package] только обычные файлы и соответствующие каталоги. Пути — относительные POSIX, [UTF-8][g-identifier-formats]; slug — `[a-z0-9][a-z0-9_-]*`. Разрешены корневые три файла, `skills/<slug>/SKILL.md` и объявленные файлы под `skills/<slug>/resources/`. Нет `..`, `.`, пустых сегментов, обратного слеша, абсолютного/drive/UNC пути, control chars, NUL, symlink/hardlink/device entries, дубликатов и Unicode/case-fold коллизий. Archive path и manifest path должны совпасть после единственной принятой NFC-нормализации; конфликт отклоняется, а не «исправляется».

Лимиты [SRS][g-requirement-ids]: вход [ZIP][g-package] ≤10 MiB (10 485 760 байт), фактически распакованные данные суммарно ≤25 MiB (26 214 400 байт), ≤1000 обычных файлов включая manifest. Счётчики проверяются потоково, а не по доверенным [ZIP][g-package] headers. Зашифрованные/многотомные [ZIP][g-package] не поддерживаются. Resources — обычные объявленные файлы, обрабатываемые как инертные байты; вложенные архивы не распаковываются и не обходят суммарные лимиты. На лимите значение допускается, превышение на один байт/файл отвергается. [Staging][g-staging] не имеет executable permissions.

<a id="manifest"></a>
## 3. Manifest v1

Пример использует символические [SHA-256][g-config-hash] placeholders; реальные значения — 64 lowercase hex, вычисляются по байтам файлов. В реальном [пакете][g-package] placeholders недопустимы. `files` перечисляет все файлы, кроме самого manifest, чтобы не образовывать рекурсивный hash. [SHA-256][g-config-hash] удостоверяет целостность относительно manifest, а не доверенного автора; подписанные [пакеты][g-package] не заявлены.

```json
{
  "format": "portway.agent-package",
  "format_version": 1,
  "agent": {
    "name": "Demo coder",
    "role_code": "coder",
    "role_catalog_version": 1,
    "instructions_file": "AGENTS.md",
    "skills": [{"alias":"project_rules","file":"skills/project_rules/SKILL.md"}],
    "model_alias": "primary_model",
    "tool_aliases": ["workspace_read", "workspace_write", "shell"],
    "mcp_aliases": ["documentation"],
    "agent_aliases": [{"alias":"review_partner","role_code":"reviewer"}],
    "policy": {"workspace_read":"allow","workspace_write":"allow","shell":"approval"},
    "limits": {"run_seconds":900,"steps":20,"tokens":20000,"output_tokens":2048}
  },
  "requirements": {
    "models": [{"alias":"primary_model","capabilities":["streaming","tool_calling"],"usage_mode":"reported_or_conservative"}],
    "tools": [{"alias":"workspace_read","kind":"workspace.read"},{"alias":"workspace_write","kind":"workspace.write"},{"alias":"shell","kind":"shell.exec"}],
    "secrets": [{"alias":"docs_credential","purpose":"documentation_adapter"}]
  },
  "files": [
    {"path":"AGENTS.md","bytes":120,"sha256":"SHA256_OF_AGENTS"},
    {"path":"mcp.json","bytes":180,"sha256":"SHA256_OF_MCP"},
    {"path":"skills/project_rules/SKILL.md","bytes":80,"sha256":"SHA256_OF_SKILL"}
  ]
}
```

| Поле | Тип / валидация |
|---|---|
| `format`, `format_version` | Точное имя и integer 1; другая версия→415, не best-effort [импорт][g-export-import] |
| `agent.name`, `role_code`, `role_catalog_version` | Непустые строки/версия каталога; [роль][g-role] должна существовать или быть явно сопоставлена совместимому preset |
| `instructions_file` | Ровно AGENTS.md, валидный [UTF-8][g-identifier-formats] |
| `skills` | Уникальные alias/file; каждый SKILL.md присутствует, resources входят в files |
| `model_alias`, `tool_aliases`, `mcp_aliases`, `agent_aliases` | Уникальные символические ссылки; ни local [UUID][g-identifier-formats], ни host path |
| `policy` | Только известные capabilities и allow/[approval][g-approval]/deny; нет server-wide override |
| `limits` | Положительные integers в пределах [SRS][g-requirement-ids]/server profile; запрос расширения не активируется молча |
| `requirements.models` | Требуемые capabilities; label [модели][g-model] может быть подсказкой, но endpoint/credential не импортируются автоматически |
| `requirements.tools` | Alias→зарегистрированный kind; не shell-команда для установки [tool][g-tool] |
| `requirements.secrets` | Alias+purpose; никакого значения, encrypted blob или source secret ID |
| `files` | Уникальные допустимые пути, integer bytes≥0, [SHA-256][g-config-hash]; полное совпадение фактического состава/размеров/hash |

Во всех JSON-объектах неизвестные поля и повторяющиеся ключи отклоняются; parser не принимает последнюю копию дублированного ключа. Alias сравнивается точно после принятой нормализации. JSON не содержит комментариев. AGENTS.md/SKILL.md — [UTF-8][g-identifier-formats] текст; их содержимое — недоверенные инструкции, а не полномочия.

<a id="mcp"></a>
## 4. [MCP][g-mcp] и внешние привязки

```json
{
  "schema_version": 1,
  "servers": [{
    "alias": "documentation",
    "adapter_kind": "docs.lookup",
    "endpoint_alias": "docs_endpoint",
    "secret_alias": "docs_credential",
    "capabilities": ["read_documentation"]
  }]
}
```

Endpoint alias выбирает configured allowlisted adapter цели, а не произвольный URL из [пакета][g-package]. Secret alias сопоставляется с доступной [владельцу][g-owner] server [secret reference][g-secret-reference]. Local launch command, package executable, arbitrary env values с credential и absolute paths запрещены. Настройки [tools][g-tool] — декларативные kind/нужные capabilities; выбранный target adapter и его ограничения сохраняются в локальном snapshot.

[Agent][g-agent] aliases связываются с конкретной доступной [AgentVersion][g-agent-version]/[ролью][g-role] на цели. [Импорт][g-export-import] версии не включает новые глобальные links/[hooks][g-hook] автоматически: [владелец][g-owner] проверяет направленную топологию отдельно. Объявленный required alias без [mapping][g-staging] блокирует [run][g-run], даже если сам [AgentVersion][g-agent-version] уже доступен для чтения.

<a id="normalization"></a>
## 5. Нормализованная конфигурация и сравнение

Canonical portable config состоит из нормализованных текстов инструкций/[skills][g-skill], ordered [tool][g-tool]/[MCP][g-mcp]/agent-alias declarations, [role][g-role]/catalog version, requested [policy][g-policy], limits и [model][g-model] capabilities. JSON object keys сортируются, массивы множеств сортируются по alias, порядок смыслового текста сохраняется; текст [UTF-8][g-identifier-formats]/NFC с LF. `config_hash` — [SHA-256][g-config-hash] этой canonical JSON. File hashes считаются по исходным байтам отдельно; разные [ZIP][g-package] timestamps/compression не означают другую конфигурацию.

В hash не входят source installation/user/[agent][g-agent] [UUID][g-identifier-formats], [export][g-export-import] time, имя скачанного файла, target endpoint/[model][g-model] ID/secret ID и локальные пути. Сопоставление M1→M2 меняет binding, а не инструкции/права/[tools][g-tool]/[skills][g-skill]. Серверные target restrictions вычисляются отдельно и показываются как effective-policy diff; несовместимость, потеря capability или существенное сужение, мешающее контрольной [задаче][g-task], не выдается за доказанный перенос. Source и target portable config сравниваются по согласованным полям и hash; functional result проверяется отдельно, текст [модели][g-model] не обязан совпасть.

<a id="pipeline"></a>
## 6. [Export][g-export-import] и безопасный [import][g-export-import] pipeline

| [Шаг][g-step] | Действие | Ошибка / атомарность |
|---|---|---|
| [Export][g-export-import] | Прочитать выбранную immutable версию; заменить local bindings aliases; построить files/hashes и [ZIP][g-package] | Отказ при непредставимых/секретных полях; не выдавать урезанный «успех» |
| Upload | [Owner][g-owner]/device guard, [streaming][g-model-protocol] входной cap, изолированный [staging][g-staging] handle | Ошибка не создаёт [AgentVersion][g-agent-version]; после upload ответа доступен import_id |
| Container validation | [ZIP][g-package] entry/path/type/collision checks, фактические byte/file counters | Traversal/bomb/symlink/duplicate→INVALID_PACKAGE/413; никакой записи вне [staging][g-staging] |
| Content validation | Manifest/schema/version/files/hash/[UTF-8][g-identifier-formats], JSON duplicate keys, запрещённые bindings | Несоответствие→reject, без исполнения и регистрации [инструментов][g-tool] |
| [Mapping][g-staging] preview | [Model][g-model]/[provider][g-provider], secret refs, [tool][g-tool] adapters, [MCP][g-mcp] endpoint, [agent][g-agent] aliases, [role][g-role] version | Список [unresolved][g-staging] и effective-policy diff; source IDs не принимаются как готовые target IDs |
| Validate [mappings][g-staging] | Права [owner][g-owner] на target refs, network allowlist, capabilities и лимиты | Неподтверждённая совместимость блокирует [run][g-run]/[activation][g-staging] согласно required [mapping][g-staging] |
| Activate | Под lock [import][g-export-import] version: конфигурация+[AgentVersion][g-agent-version]+audit+activated reference | Одна [транзакция][g-atomicity]; same key возвращает тот же [AgentVersion][g-agent-version]; [rollback][g-runbook] не оставляет частично активный [агент][g-agent] |
| Cleanup | Удалить rejected/cancelled/expired [staging][g-staging]; отдельно retain [evidence][g-evidence] [ZIP][g-package] пользователя | Ошибка cleanup видна; не удалять выбранный пользователем [export][g-export-import] |

Основные состояния [import][g-export-import]: `uploaded→validated→mapping_required|ready→activated`, ошибки `rejected/cancelled/expired`. Активировать можно только [ready][g-readiness]; required [mappings][g-staging] полностью разрешены. Временный [staging][g-staging] имеет срок по server configuration, выдаваемый `expires_at`; в этом baseline не вводится новый скрытый product SLA. Истёкший [import][g-export-import]→410 и повтор upload, не auto-activation старого [ZIP][g-package].

Клиент никогда не хранит распакованный [пакет][g-package] как доверенную конфигурацию. Сервер — authority validation/[mapping][g-staging]/[activation][g-staging]; [Android][g-flutter] показывает server result, имя/размер/[роль][g-role]/права/зависимости и явное действие «Создать [агента][g-agent]». Восстановление после потери ответа — GET [import][g-export-import] и повтор исходного ключа. Cancel picker/upload не создаёт [agent][g-agent].

<a id="portability"></a>
## 7. Обязательный протокол переноса

| Ячейка | Что закрепить до [запуска][g-run] | Проверка |
|---|---|---|
| A/M1 | Host A, [model][g-model] M1, [AgentVersion][g-agent-version]/config hash, source commit, контрольная [задача][g-task] | Настоящий [tools][g-tool]/result, заранее заданный functional assertion |
| A/M2 | [ZIP][g-package] с телефона, новый [model][g-model]/[provider][g-provider] binding, portable config comparison | Та же [задача][g-task]/ревизия, M2 capabilities, тот же functional assertion |
| B/M2 | Отдельные [control plane][g-control-plane]/[Runtime][g-runtime] на host/VM B, [импорт][g-export-import] с телефона, новые secrets/paths/aliases | Новый [run][g-run] ID, идентичные agreed portable fields, functional assertion |

Для каждой ячейки сохранить installation ID, описание отдельного host/VM, [model][g-model] ID/endpoint label без ключа, версии сервера/[APK][g-apk], [AgentVersion][g-agent-version]/config hash, source/result revision, [run][g-run]/[chain][g-chain] IDs, assertions и outcome. Хотя бы одна [модель][g-model] локальная. Смена каталога на A не заменяет B, одно распаковывание не заменяет [запуск][g-run]. [Runner][g-runner] и [GitHub][g-github] PR для этого не нужны. Подробные [TC-015][id-tc-015]/016/026 и измерения [MX-06][id-mx-06] — [DOC-057][id-doc-057]/[SMART][g-smart]; сейчас все NOT_RUN.

## История

| Версия | Дата | Изменение |
|---|---|---|
| v0.1 | 10.10.2026 | Заданы формат [ZIP][g-package]/JSON, validation/[mapping][g-staging]/[activation][g-staging] и доказательство переноса |

[g-agent]: 07-glossary.md#term-agent

[g-agent-version]: 07-glossary.md#term-agent-version

[g-apk]: 07-glossary.md#term-apk

[g-approval]: 07-glossary.md#term-approval

[g-atomicity]: 07-glossary.md#term-atomicity

[g-chain]: 07-glossary.md#term-chain

[g-config-hash]: 07-glossary.md#term-config-hash

[g-control-plane]: 07-glossary.md#term-control-plane

[g-device-session]: 07-glossary.md#term-device-session

[g-evidence]: 07-glossary.md#term-evidence

[g-export-import]: 07-glossary.md#term-export-import

[g-flutter]: 07-glossary.md#term-flutter

[g-github]: 07-glossary.md#term-github

[g-hook]: 07-glossary.md#term-hook

[g-host]: 07-glossary.md#term-host

[g-identifier-formats]: 07-glossary.md#term-identifier-formats

[g-mcp]: 07-glossary.md#term-mcp

[g-model]: 07-glossary.md#term-model

[g-model-protocol]: 07-glossary.md#term-model-protocol

[g-owner]: 07-glossary.md#term-owner

[g-package]: 07-glossary.md#term-package

[g-policy]: 07-glossary.md#term-policy

[g-portability]: 07-glossary.md#term-portability

[g-provider]: 07-glossary.md#term-provider

[g-push]: 07-glossary.md#term-push

[g-readiness]: 07-glossary.md#term-readiness

[g-requirement-ids]: 07-glossary.md#term-requirement-ids

[g-role]: 07-glossary.md#term-role

[g-run]: 07-glossary.md#term-run

[g-runbook]: 07-glossary.md#term-runbook

[g-runner]: 07-glossary.md#term-runner

[g-runtime]: 07-glossary.md#term-runtime

[g-secret-reference]: 07-glossary.md#term-secret-reference

[g-skill]: 07-glossary.md#term-skill

[g-smart]: 07-glossary.md#term-smart

[g-staging]: 07-glossary.md#term-staging

[g-step]: 07-glossary.md#term-step

[g-task]: 07-glossary.md#term-task

[g-tool]: 07-glossary.md#term-tool

[g-workspace]: 07-glossary.md#term-workspace

[id-d-11]: 02-technical-specification.md#d-11

[id-doc-038]: 38-agent-package-format.md

[id-doc-057]: 57-verification-and-traceability.md

[id-fr-060]: 03-software-requirements-specification.md#fr-060

[id-mx-06]: 04-smart-goals-and-metrics.md#mx-06

[id-tc-015]: 57-verification-and-traceability.md#tc-015
