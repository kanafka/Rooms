[![Review Assignment Due Date](https://classroom.github.com/assets/deadline-readme-button-22041afd0340ce965d47ae6ef1cefeee28c7c493a6346c4f15d667ab976d596c.svg)](https://classroom.github.com/a/uvnTmvcw)

# Room Booking Service

Сервис бронирования переговорок на Go.

## Быстрый старт

```bash
# Запуск со всеми зависимостями
make up

# Наполнение БД тестовыми данными
make seed

# Запуск тестов
make test

# Линтер
make lint
```

## Архитектурные решения

### Генерация слотов

**Подход: ленивая генерация (lazy, on-demand) с хранением в БД.**

При запросе `GET /rooms/{roomId}/slots/list?date=YYYY-MM-DD`:
1. Проверяется, входит ли запрошенная дата в расписание комнаты (день недели).
2. Вычисляются все 30-минутные слоты для этой даты.
3. Слоты записываются в БД через `INSERT ... ON CONFLICT (room_id, start_time) DO NOTHING` — это обеспечивает идемпотентность и отсутствие дублей.
4. UUID слота детерминирован: `uuid.NewSHA1(uuid.NameSpaceURL, []byte(roomID+"|"+slotStart))`. Это гарантирует, что один и тот же слот всегда получает один и тот же UUID, независимо от того, сколько раз вызывается генерация.
5. Возвращаются только слоты без активных броней (LEFT JOIN bookings WHERE active IS NULL).

**Почему именно так:**
- Слоты имеют стабильные UUID в БД — необходимо для бронирования по `slotId`.
- Не нужен фоновый cron-job для генерации слотов.
- Операция идемпотентна: повторный вызов не создаёт дублей.
- Самый нагруженный эндпоинт оптимизирован: индекс `idx_slots_room_start` и `idx_bookings_slot_status` обеспечивают быстрый поиск.
- В 99.9% случаев запросы — ближайшие 7 дней, и при первом запросе слоты кешируются в БД.

### Ссылка на конференцию (Conference Link)

**Принятые решения при сбоях:**

| Сценарий | Поведение |
|----------|-----------|
| Внешний сервис недоступен | Бронь создаётся успешно, `conferenceLink = null`, ошибка логируется |
| Внешний сервис вернул ошибку | Бронь создаётся успешно, `conferenceLink = null`, ошибка логируется |
| Ошибка сохранения ссылки в БД после успешного ответа | Бронь создаётся успешно, `conferenceLink = null`, ошибка логируется |
| Успех | Ссылка сохраняется в поле `conference_link` брони |

**Почему:** Операция получения ссылки на конференцию вторична по отношению к самому факту бронирования. Сбой внешнего сервиса не должен делать бронирование недоступным. Пользователь получает бронь в любом случае.

Мок-сервис реализован в `internal/usecase/conference.go`. В реальной системе это был бы HTTP-клиент с таймаутом и circuit breaker.

### Аутентификация

- `POST /dummyLogin` — возвращает JWT с фиксированным UUID для каждой роли (не сохраняет в БД).
  - admin UUID: `00000000-0000-0000-0000-000000000001`
  - user UUID: `00000000-0000-0000-0000-000000000002`
- `POST /register` / `POST /login` — полноценная регистрация и вход с bcrypt-хешем пароля.
- JWT содержит `user_id` (UUID) и `role`.

### Архитектура

```
cmd/server/        — точка входа
docs/              — сгенерированная Swagger-документация (swag init)
internal/
  config/          — конфигурация из переменных окружения
  domain/          — модели и ошибки предметной области
  repository/      — интерфейсы репозиториев
    postgres/      — реализации на pgx/v5
  usecase/         — бизнес-логика
  delivery/        — HTTP-обработчики (chi router)
  e2e/             — E2E тесты (требуют TEST_DATABASE_URL)
migrations/        — SQL миграции
seed/              — заполнение БД тестовыми данными
```

Слои зависят только вниз: delivery → usecase → repository. Usecase не знает о HTTP.

## API Endpoints

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| GET | /_info | — | Health check, всегда 200 |
| POST | /dummyLogin | — | Получить JWT по роли |
| POST | /register | — | Регистрация пользователя |
| POST | /login | — | Вход по email/паролю |
| GET | /rooms/list | any | Список переговорок |
| POST | /rooms/create | admin | Создать переговорку |
| POST | /rooms/{roomId}/schedule/create | admin | Создать расписание |
| GET | /rooms/{roomId}/slots/list | any | Свободные слоты на дату |
| POST | /bookings/create | user | Создать бронь |
| GET | /bookings/list | admin | Все брони (пагинация) |
| GET | /bookings/my | user | Мои будущие брони |
| POST | /bookings/{bookingId}/cancel | user | Отменить бронь |

## Конфигурация

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/booking?sslmode=disable` | URL PostgreSQL |
| `JWT_SECRET` | `secret` | Секрет для подписи JWT |
| `PORT` | `8080` | Порт HTTP-сервера |
| `ADMIN_UUID` | `00000000-0000-0000-0000-000000000001` | Фиксированный UUID для admin в dummyLogin |
| `USER_UUID` | `00000000-0000-0000-0000-000000000002` | Фиксированный UUID для user в dummyLogin |

## Тесты

```bash
# Юнит-тесты
go test ./internal/usecase/... ./internal/delivery/...

# E2E тесты (нужна тестовая БД)
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/booking_test?sslmode=disable \
  go test -tags e2e ./internal/e2e/...

# Все тесты
make test
```

## Вопросы по заданию и принятые решения

**1. Что делать, если запрашивается дата в прошлом для слотов?**
Слоты генерируются и возвращаются (пустые, если не в расписании). Создание брони на прошедший слот возвращает 400.

**2. Что возвращать в `/bookings/my` если нет броней?**
Пустой массив `{"bookings": []}`.

**3. Что делать если бронь уже отменена и пользователь снова вызывает cancel?**
Идемпотентно: возвращается 200 с текущим состоянием брони (status: cancelled).
