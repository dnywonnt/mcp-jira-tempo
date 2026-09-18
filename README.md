# mcp-jira-tempo

MCP-сервер для работы с Jira и Tempo Timesheets в self-hosted Jira Data Center.

Сервер предоставляет инструменты для просмотра задач Jira, проверки подключения к Tempo и логирования времени в Tempo.

## Возможности

- Получение текущего пользователя Jira и выбранного Tempo worker.
- Получение задачи Jira по ключу.
- Поиск задач Jira по JQL.
- Список нерешенных задач, назначенных на текущего пользователя.
- Проверка доступности Jira и Tempo endpoint.
- Логирование времени в Tempo для одной задачи.
- Массовое логирование времени в Tempo для нескольких задач.
- Режим `dryRun` для проверки payload без создания worklog.

## Требования

- Go `1.27+`.
- Jira Data Center с REST API `/rest/api/2`.
- Tempo Timesheets endpoint `/rest/tempo-timesheets/4/worklogs`.
- Jira token с доступом к задачам и созданию Tempo worklog.

## Конфигурация

Минимальная конфигурация для одного Jira instance:

```bash
export JIRA_BASE_URL="https://jira.example.test"
export JIRA_TOKEN="your-token"
```

Запуск:

```bash
go run ./cmd
```

### Переменные окружения

| Переменная | Обязательная | Описание |
|---|---:|---|
| `JIRA_BASE_URL` | да | Base URL Jira для default instance. |
| `JIRA_TOKEN` | да | Bearer token для Jira/Tempo API. |
| `JIRA_WORKER` | нет | Tempo worker. Если не задан, используется текущий Jira user. |
| `JIRA_WORKER_FIELD` | нет | Поле пользователя для worker: `key` или `name`. По умолчанию `key`. |
| `JIRA_BILLABLE_BY_DEFAULT` | нет | Если `true`, `billableSeconds` по умолчанию равен `seconds`. |
| `JIRA_HTTP_CLIENT_TIMEOUT` | нет | Таймаут HTTP-клиента, например `45s`. По умолчанию `30s`. |
| `JIRA_MAX_RESPONSE_BYTES` | нет | Максимальный размер ответа API. По умолчанию `1048576`. |

## Несколько Jira Instance

Можно настроить несколько инстансов через `JIRA_INSTANCES`.

```bash
export JIRA_INSTANCES="bk,corp"
export JIRA_DEFAULT_INSTANCE="bk"

export JIRA_BK_BASE_URL="https://bk-jira.example.test"
export JIRA_BK_TOKEN="bk-token"
export JIRA_BK_WORKER="JIRAUSER123"
export JIRA_BK_WORKER_FIELD="key"

export JIRA_CORP_BASE_URL="https://corp-jira.example.test"
export JIRA_CORP_TOKEN="corp-token"
export JIRA_CORP_BILLABLE_BY_DEFAULT="true"
```

Для alias переменные строятся как `JIRA_<ALIAS>_<SUFFIX>`. Символы кроме букв и цифр заменяются на `_`, регистр приводится к upper case.

## Инструменты MCP

### `jira_whoami`

Проверяет текущего Jira пользователя и выбранного Tempo worker.

```json
{
  "instance": "bk"
}
```

### `jira_get_issue`

Возвращает задачу Jira по ключу: `id`, `key`, `summary`, `description`, `status`.

```json
{
  "instance": "bk",
  "issueKey": "APPSUP-1575"
}
```

### `jira_list_issues`

Ищет задачи по JQL. Если `jql` не задан, используется:

```sql
assignee = currentUser() AND resolution = Unresolved ORDER BY updated DESC
```

```json
{
  "instance": "bk",
  "jql": "project = APPSUP AND resolution = Unresolved ORDER BY updated DESC",
  "maxResults": 50
}
```

`maxResults` ограничен значением `100`.

### `tempo_health_check`

Проверяет доступность Jira `/myself` и Tempo worklogs endpoint.

```json
{
  "instance": "bk"
}
```

### `tempo_log_time`

Создает один Tempo worklog.

```json
{
  "instance": "bk",
  "issueKey": "APPSUP-1575",
  "date": "2026-09-18",
  "seconds": 300,
  "comment": "тест",
  "dryRun": false
}
```

Опционально можно передать `billableSeconds`:

```json
{
  "instance": "bk",
  "issueKey": "APPSUP-1575",
  "date": "2026-09-18",
  "seconds": 3600,
  "billableSeconds": 1800,
  "comment": "разработка"
}
```

### `tempo_log_time_bulk`

Создает несколько Tempo worklog последовательно.

Если одна запись завершилась ошибкой, остальные продолжают выполняться. Ответ содержит счетчики `succeeded`, `failed` и результат по каждой entry.

```json
{
  "instance": "bk",
  "dryRun": false,
  "entries": [
    {
      "issueKey": "APPSUP-1575",
      "date": "2026-09-18",
      "seconds": 300,
      "comment": "тест"
    },
    {
      "issueKey": "DEV-5175",
      "date": "2026-09-18",
      "seconds": 1800,
      "comment": "разработка"
    }
  ]
}
```

## Dry Run

Для `tempo_log_time` и `tempo_log_time_bulk` можно указать `dryRun: true`.

В этом режиме сервер вернет сформированный Tempo request, но не создаст worklog.

## Разработка

Запуск тестов:

```bash
go test ./...
```

Генерация моков:

```bash
go generate ./internal/mcp
```

## Лицензия

MIT.
