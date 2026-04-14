# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Docker и Docker Compose

## Быстрый запуск

```bash
docker compose up --build
```

После запуска сервис доступен на `http://localhost:8080`.

Если postgres уже запускался ранее — пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

## Swagger UI
http://localhost:8080/swagger/

## API

Базовый префикс: `/api/v1`

### Задачи (существующий функционал)

| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/tasks` | Создать задачу |
| `GET` | `/tasks` | Список задач |
| `GET` | `/tasks/{id}` | Получить задачу по ID |
| `PUT` | `/tasks/{id}` | Обновить задачу |
| `DELETE` | `/tasks/{id}` | Удалить задачу |

### Периодичность (новый функционал)

| Метод | Путь | Описание |
|-------|------|----------|
| `PUT` | `/tasks/{id}/recurrence` | Установить или обновить правило |
| `GET` | `/tasks/{id}/recurrence` | Получить правило |
| `DELETE` | `/tasks/{id}/recurrence` | Удалить правило |
| `POST` | `/tasks/{id}/recurrence/generate` | Сгенерировать задачи на 30 дней вперёд |

---

## Реализация периодических задач

### Архитектура

Выбран **lazy-подход**: задачи генерируются явно по запросу `POST /recurrence/generate`, а не в фоне. Это упрощает деплой, делает поведение предсказуемым и легко тестируемым. Фоновый планировщик можно добавить поверх — он будет просто вызывать `GenerateAll` раз в сутки.

### Почему не cron с самого начала?

| Подход | Плюсы | Минусы |
|--------|-------|--------|
| Lazy (выбрано) | Простой деплой, идемпотентно, легко тестировать | Задачи появляются только после явного вызова |
| Cron | Задачи всегда готовы заранее | Нужен отдельный процесс, сложнее деплой |
| Гибрид | Лучший UX | Сложнее кодовая база |

### Почему правило в отдельной таблице, а не поле в tasks?

Хранение правила в `recurrence_rules` даёт чистое разделение ответственности: задача остаётся задачей, правило — отдельной сущностью. Можно удалить правило без удаления задачи и расширять модель правил не трогая таблицу `tasks`.

### Типы периодичности

| rule_type | Обязательные поля | Пример |
|-----------|-------------------|--------|
| `daily` | `interval_days` (минимум 1) | каждые 2 дня |
| `monthly` | `month_day` (от 1 до 30) | 15-го числа каждого месяца |
| `specific_dates` | `specific_dates` — массив дат | только указанные даты |
| `even_odd` | `day_parity`: even или odd | только чётные дни месяца |

### Модель данных

Таблица `tasks` — расширена двумя полями:
- `parent_task_id` — ссылка на задачу-шаблон (NULL у шаблонов, заполнен у дочерних)
- `scheduled_date` — дата, на которую сгенерирована задача

Таблица `recurrence_rules` — правило периодичности (один к одному с задачей-шаблоном):
- `rule_type`, `interval_days`, `month_day`, `specific_dates[]`, `day_parity`
- `start_date`, `end_date`

Таблица `recurrence_occurrences` — журнал уже сгенерированных дат:
- `UNIQUE(rule_id, scheduled_date)` — защита от дублей

### Граничные случаи

- **Февраль и короткие месяцы** — `monthly` с `month_day=30`: Go нормализует дату, код проверяет что месяц не изменился и пропускает несуществующий день
- **Изменение правила** — `PUT /recurrence` делает upsert, уже созданные задачи остаются, новые генерируются по новому правилу
- **Удаление родительской задачи** — `ON DELETE CASCADE` удаляет правило и записи в occurrences; дочерние задачи получают `parent_task_id = NULL`
- **Повторный вызов generate** — `ON CONFLICT DO NOTHING` плюс `UNIQUE` constraint гарантируют идемпотентность
- **Сервер был выключен** — генерация стартует с текущей даты при следующем вызове, прошедшие даты не создаются

### Расширяемость

Чтобы добавить тип `weekly` (по дням недели): реализовать метод `weeklyOccurrences` в `Rule`, добавить константу `RuleTypeWeekly` и поле `weekdays INT[]` в новой миграции. Остальной код менять не нужно.

---

## Примеры запросов

### Создать задачу-шаблон

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Обход пациентов","description":"Ежедневный обход"}'
```

### Установить правило — каждые 2 дня

```bash
curl -X PUT http://localhost:8080/api/v1/tasks/1/recurrence \
  -H "Content-Type: application/json" \
  -d '{"rule_type":"daily","interval_days":2,"start_date":"2026-04-14"}'
```

### Установить правило — 15-го числа каждого месяца

```bash
curl -X PUT http://localhost:8080/api/v1/tasks/1/recurrence \
  -H "Content-Type: application/json" \
  -d '{"rule_type":"monthly","month_day":15,"start_date":"2026-04-01","end_date":"2026-12-31"}'
```

### Установить правило — конкретные даты

```bash
curl -X PUT http://localhost:8080/api/v1/tasks/1/recurrence \
  -H "Content-Type: application/json" \
  -d '{"rule_type":"specific_dates","specific_dates":["2026-05-01","2026-05-10","2026-05-25"],"start_date":"2026-05-01"}'
```

### Установить правило — чётные дни месяца

```bash
curl -X PUT http://localhost:8080/api/v1/tasks/1/recurrence \
  -H "Content-Type: application/json" \
  -d '{"rule_type":"even_odd","day_parity":"even","start_date":"2026-04-14"}'
```

### Сгенерировать задачи на 30 дней вперёд

```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/recurrence/generate
```

### Посмотреть правило

```bash
curl http://localhost:8080/api/v1/tasks/1/recurrence
```

### Удалить правило

```bash
curl -X DELETE http://localhost:8080/api/v1/tasks/1/recurrence
```

### Пример ответа при генерации

```json
[
  {
    "id": 2,
    "title": "Обход пациентов",
    "description": "Ежедневный обход",
    "status": "new",
    "parent_task_id": 1,
    "scheduled_date": "2026-04-14T00:00:00Z",
    "created_at": "2026-04-14T10:00:00Z",
    "updated_at": "2026-04-14T10:00:00Z"
  }
]
```

---

## Допущения

- Горизонт генерации — 30 дней вперёд от текущей даты
- Все даты хранятся и обрабатываются в UTC
- Задача-шаблон остаётся видимой в общем списке задач
- При `month_day=31` в месяцах с 30 днями задача в этом месяце не создаётся
- Нечётные и чётные дни определяются по числу месяца: 1, 3, 5 — нечётные; 2, 4, 6 — чётные