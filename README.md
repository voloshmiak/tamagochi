# Тамагочі

Веб-гра з віртуальним улюбленцем: догляд з браузера або з головного екрана телефона (PWA), характер улюбленця на основі utility AI, друзі, рейтинг, обмін предметами та репліки улюбленця через LLM.

Проєкт виконується в межах дисципліни «Життєвий цикл програмного забезпечення» (КПІ ім. Ігоря Сікорського, ФІОТ).

> **Стан:** інтегрована система (лабораторна робота №5). Компоненти спринту 1 з'єднані в одне ціле: фронтенд → Caddy (HTTPS) → Go API → PostgreSQL і Redis. Дані зберігаються в PostgreSQL, схема створюється версійованими міграціями goose під час старту API, refresh-сесії лежать у Redis, листи підтвердження email надсилаються через SMTP (локально це Mailpit). Контракт API описано в OpenAPI: з нього генеруються інтерфейс Go-сервера, TS-типи клієнта, і за ним перевіряється кожен запит. Модулі наступних спринтів (`social`, `trade`, `leaderboard`, `notify`, `replica`) поки мають лише сутності, порти й сигнатури сервісів; їхні методи повертають `apperr.ErrNotImplemented`.

## Стек

| Рівень | Технології |
|---|---|
| Бекенд | Go 1.26, chi, oapi-codegen + kin-openapi (валідація за OpenAPI), JWT (golang-jwt), bcrypt |
| Дані | PostgreSQL 17 (pgx/v5, sqlc, міграції goose), Redis 7 (go-redis: refresh-сесії) |
| Фронтенд | React 18, Vite, TypeScript, Tailwind, TanStack Query, Zustand, i18next, openapi-fetch + openapi-typescript |
| Інфраструктура | Docker Compose, Caddy 2 (HTTPS, HSTS), Mailpit (локальна пошта), GitHub Actions |
| Заплановано | asynq, WebSocket, Web Push |

## Архітектура

Модульний моноліт на Go плюс окремий процес `pet-engine`, що збирається з того самого репозиторію (документ «Архітектура програмного забезпечення», лабораторна робота №2).

Кожен модуль — окремий Go-пакет із доменними типами, портами (інтерфейсами) та сервісом. Модулі не звертаються до сховищ або сервісів напряму, а отримують залежності через інтерфейси. Конкретні реалізації створюються лише в composition root [internal/app](internal/app/app.go).

## Структура проєкту

```
tamagochi/
├── cmd/
│   ├── api/                 # точка входу HTTP-сервера
│   ├── migrate/             # застосування міграцій без запуску API
│   └── pet-engine/          # точка входу фонового воркера (LLM, push, обміни)
├── internal/
│   ├── api/oapi/            # код, згенерований з api/openapi.yaml (oapi-codegen)
│   ├── app/                 # composition root: вибір адаптерів, роутер
│   ├── config/              # налаштування зі змінних середовища
│   ├── auth/                # реєстрація, вхід, JWT        (FR-AUTH)
│   ├── pets/                # улюбленець, дії, інвентар     (FR-PET)
│   ├── brain/               # utility AI та риси характеру  (FR-AI)
│   ├── social/              # друзі, блокування             (FR-SOC)
│   ├── trade/               # обмін предметами              (FR-TR)
│   ├── leaderboard/         # очки та рейтинги              (FR-LB)
│   ├── notify/              # Web Push, тихі години         (FR-NOT)
│   ├── replica/             # репліки улюбленця через LLM   (FR-LLM)
│   ├── realtime/            # контракт WebSocket/pub-sub    (FR-SOC-04)
│   ├── queue/               # контракт черги задач API → pet-engine
│   └── platform/            # спільні компоненти
│       ├── apperr/          #   категорії помилок
│       ├── clock/           #   абстракція часу
│       ├── httpx/           #   JSON, мапінг помилок, middleware автентифікації
│       ├── id/              #   генерація ідентифікаторів
│       ├── pg/              #   PostgreSQL: пул, TxManager, міграції, запити sqlc
│       └── tx/              #   інтерфейс TxManager (межа транзакції)
├── api/openapi.yaml         # контракт API (NFR-MNT-01)
├── deploy/                  # Caddyfile і образ Caddy зі зібраним фронтендом
├── web/                     # фронтенд (React + Vite + TS)
├── docker-compose.yml       # уся система однією командою (NFR-MNT-06)
├── sqlc.yaml                # конфігурація генерації коду з SQL
├── .env.example             # приклад змінних середовища
├── .gitignore
├── LICENSE
└── Makefile
```

### Основні класи та їх відповідальність

| Модуль | Тип | Відповідальність | Стан |
|---|---|---|---|
| auth | `User` | дані облікового запису | ✅ |
| auth | `Service` | `Register`, `Login`, `Refresh` | ✅ |
| auth | `UserRepository`, `VerificationRepository`, `RefreshStore`, `PasswordHasher`, `TokenIssuer`, `Mailer` | порти | ✅ + PostgreSQL, Redis, bcrypt, JWT, SMTP та in-memory адаптери |
| pets | `Pet` | `Recompute` (ліниве обчислення), `Apply`, `CooldownLeft` | ✅ |
| pets | `ActionRule` / `StatRule` | правила дій (кулдаун, ефект, предмет, очки) | ✅ |
| pets | `Inventory`, `Item`, `Catalog` | інвентар, каталог предметів, щоденна нагорода | ✅ |
| pets | `Stage` | стадія розвитку за віком улюбленця | ✅ |
| pets | `Service` | `Create`, `Get`, `Perform`, `ClaimDailyReward` у транзакції | ✅ |
| pets | `PetRepository`, `InventoryRepository`, `Decider` | порти | ✅ + PostgreSQL (`PgPetRepository`, `PgInventoryRepository`) та in-memory адаптери |
| platform/pg | `TxManager` | транзакція pgx у контексті; репозиторії приєднуються до неї | ✅ |
| brain | `UtilityBrain`, `Consideration` (+ 4 реалізації), `Traits` | вибір поведінки, детермінізм через seed | ✅ |
| trade | `Offer` | пропозиція обміну та її стани | ✅ |
| trade | `Service`, `InventoryPort`, `IdempotencyStore` | атомарний обмін | заглушки |
| social | `Relation`, `Service`, `Repository` | дружба, блокування | заглушки |
| leaderboard | `Entry`, `Service`, `Store` | рейтинги | заглушки |
| notify | `Notification`, `Subscription`, `Service`, `Sender`, `Store` | push-сповіщення | заглушки |
| replica | `Service`, `Provider` (LLMProvider), `Cache`, `Quota`, `Templates` | репліки через LLM | заглушки |
| realtime | `Broker`, `Message` | події в реальному часі | контракт |
| queue | `Task`, `Enqueuer`, `Registrar` | фонові задачі | контракт |

### Принципи SOLID у коді

- **S — єдина відповідальність.** `User` містить лише дані облікового запису, а ігрові дані зберігаються в `pets`. `Pet` відповідає за свій стан, `Service` — за сценарій використання, `Handler` — лише за HTTP.
- **O — відкритість/закритість.** Нова дія додається новим `ActionRule` у `DefaultRules()` без змін `Pet` і `Service`. Новий фактор поведінки додається новою реалізацією `Consideration` без змін `UtilityBrain`.
- **L — підстановка Лісков.** Будь-яка реалізація порту (`MemoryPetRepository` або `PgPetRepository`, `MemoryRefreshStore` або `RedisRefreshStore`) підставляється в сервіс без зміни його поведінки.
- **I — розділення інтерфейсів.** Порти вузькі: `Decider` має один метод, `TokenVerifier` у middleware — лише перевірку токена. `trade` бачить інвентар лише через `InventoryPort`, а не весь сервіс `pets`.
- **D — інверсія залежностей.** Сервіси залежать від абстракцій (`PetRepository`, `tx.Manager`, `clock.Clock`, `LLMProvider`). Конкретні адаптери підставляються в [internal/app](internal/app/app.go).

## Точки інтеграції

```
Браузер (PWA)  ──HTTPS──▶  Caddy  ──/api/*, /healthz──▶  Go API  ──▶  PostgreSQL  (користувачі, улюбленці, інвентар)
     ▲                       │                              │
     └── статика web/dist ◀──┘                              ├──▶  Redis       (refresh-сесії)
                                                            └──▶  SMTP        (листи підтвердження; локально Mailpit)
```

| Межа | Як взаємодіють | Де в коді |
|---|---|---|
| Фронтенд ↔ API | REST/JSON за контрактом [api/openapi.yaml](api/openapi.yaml). TS-типи генеруються з контракту ([web/src/api/schema.d.ts](web/src/api/schema.d.ts)), запити йдуть через `openapi-fetch` з `credentials: "include"`. На 401 клієнт сам викликає `/auth/refresh` і повторює запит | [web/src/api/client.ts](web/src/api/client.ts) |
| Caddy ↔ API | Caddy завершує TLS, віддає зібраний фронтенд і проксіює `/api/*` та `/healthz` на `api:8080` усередині мережі Compose | [deploy/Caddyfile](deploy/Caddyfile) |
| API: контракт | Маршрути беруться з `oapi.ServerInterface`, згенерованого з контракту. Кожен запит спершу проходить JWT-middleware (публічні операції визначаються за `security: []` у специфікації), потім валідацію за схемою (400 з поясненням) | [internal/app/app.go](internal/app/app.go), [internal/platform/httpx/openapi.go](internal/platform/httpx/openapi.go) |
| API ↔ PostgreSQL | pgx-пул; типобезпечні запити згенеровані sqlc з [internal/platform/pg/queries](internal/platform/pg/queries). `pg.TxManager` кладе транзакцію в контекст, а дія над улюбленцем блокує його рядок через `SELECT … FOR UPDATE` | [internal/platform/pg](internal/platform/pg), [internal/pets/postgres.go](internal/pets/postgres.go), [internal/auth/postgres.go](internal/auth/postgres.go) |
| API ↔ Redis | Refresh-сесії: кожен токен має запис з TTL, а Lua-скрипти атомарно позначають використання й відкликають усю сесію при повторному використанні токена | [internal/auth/refresh_redis.go](internal/auth/refresh_redis.go) |
| API ↔ SMTP | Після реєстрації надсилається лист із посиланням `PUBLIC_URL/api/v1/auth/verify?token=…`; без `SMTP_ADDR` посилання пишеться в лог | [internal/auth/mail.go](internal/auth/mail.go) |

Адаптери обирає лише composition root ([internal/app](internal/app/app.go)): з `DATABASE_URL` використовується PostgreSQL, з `REDIS_URL` — Redis, без них дані зберігаються в пам'яті процесу. Код модулів від цього вибору не залежить.

### Схема БД

Створюється міграціями з [internal/platform/pg/migrations](internal/platform/pg/migrations).

```mermaid
erDiagram
    users ||--o| pets : "має одного"
    users ||--o{ email_verifications : "підтверджує"
    users ||--o| inventories : "має"
    pets ||--|{ pet_traits : "характер"
    inventories ||--o{ inventory_items : "містить"
    items ||--o{ inventory_items : "предмет"

    users { text id PK; text email UK; text nickname UK; bytea password_hash; bool email_verified; timestamptz created_at }
    email_verifications { text token_hash PK; text user_id FK; timestamptz expires_at }
    pets { text id PK; text owner_id FK,UK; text name; text appearance; float satiety; float mood; float energy; float cleanliness; timestamptz born_at; timestamptz updated_at; jsonb last_actions }
    pet_traits { text pet_id PK,FK; text trait PK; float weight }
    items { text id PK; text name; text kind }
    inventories { text owner_id PK,FK; timestamptz daily_reward_at }
    inventory_items { text owner_id PK,FK; text item_id PK,FK; int quantity }
```

Показники улюбленця мають `CHECK (… BETWEEN 0 AND 100)`, кількість предметів — `CHECK (quantity > 0)`, email і нікнейм (без урахування регістру) — унікальні індекси.

## Запуск інтегрованої системи

### Однією командою (Docker Compose)

Потрібен лише Docker з Compose v2. Після клонування репозиторію:

```bash
docker compose up --build
```

| Сервіс | Що робить | Адреса з хоста |
|---|---|---|
| `caddy` | HTTPS, HSTS, статика фронтенду, проксі на API | https://localhost:8443 (http://localhost:8080 перенаправляє на HTTPS) |
| `api` | Go API; під час старту застосовує міграції | лише всередині мережі Compose |
| `postgres` | PostgreSQL 17, том `pgdata` | — |
| `redis` | Redis 7, том `redisdata` | — |
| `mailpit` | приймає листи від API і показує їх у вебінтерфейсі | http://localhost:8025 |

Коли все готово, сервіс `ready` друкує адресу застосунку. Усі змінні мають значення за замовчуванням, тож `.env` для першого запуску не потрібен. Щоб змінити порти, секрети чи домен, скопіюйте [.env.example](.env.example) у `.env`.

```bash
docker compose down        # зупинити
docker compose down -v     # зупинити й видалити дані postgres/redis
```

**HTTPS локально.** Для `localhost` Caddy випускає сертифікат власним CA, тож браузер спершу попередить про недовірений сертифікат. Можна прийняти виняток або довірити кореневий сертифікат CA Caddy: скопіюйте його командою `docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-root.crt` та імпортуйте в сховище довірених сертифікатів системи чи браузера. Для продакшну задайте `SITE_ADDRESS=ваш.домен`, `TLS_ISSUER=ваш@email` і `PUBLIC_URL=https://ваш.домен`, а також відкрийте порти 80/443: Caddy сам отримає сертифікат Let's Encrypt.

**Міграції.** API застосовує нові міграції під час кожного старту (`MIGRATE_ON_START=true`), тож порожня БД сама доходить до актуальної схеми. Advisory lock не дає двом екземплярам API мігрувати одночасно. Окремо, без запуску API: `DATABASE_URL=… make migrate`. Нова зміна схеми оформлюється новим файлом `internal/platform/pg/migrations/0000N_назва.sql` із секціями `-- +goose Up` / `-- +goose Down`; міграції, які вже потрапили в `main`, не редагуються.

Приклад запитів (`-k` через локальний сертифікат):

```bash
B=https://localhost:8443/api/v1
curl -k -X POST $B/auth/register -H 'Content-Type: application/json' -d '{"email":"a@b.co","password":"secret123","nickname":"misha"}'
# лист із посиланням підтвердження — на http://localhost:8025
curl -k -c jar -X POST $B/auth/login -H 'Content-Type: application/json' -d '{"email":"a@b.co","password":"secret123"}'
curl -k -b jar -X POST $B/pets -H 'Content-Type: application/json' -d '{"name":"Bublyk","appearance":"cat"}'
curl -k -b jar -X POST $B/pets/me/actions -H 'Content-Type: application/json' -d '{"action":"feed"}'
curl -k -b jar -X POST $B/inventory/daily-reward
```

### Локально без Docker

Потрібні Go 1.26+ і Node.js 22+ (як у CI).

```bash
# бекенд на http://localhost:8080
go run ./cmd/api

# фронтенд на http://localhost:5173 (проксіює /api на :8080)
cd web
npm ci
npm run dev
```

Без `DATABASE_URL` і `REDIS_URL` API працює з даними в пам'яті (зручно для швидкої перевірки, але дані зникають після перезапуску), а посилання для підтвердження email пише в лог. Щоб працювати з реальними сховищами, передайте адреси власних екземплярів PostgreSQL, Redis і SMTP (у `docker-compose.yml` вони не публікують порти на хост):

```bash
export DATABASE_URL='postgres://tamagochi:tamagochi@localhost:5432/tamagochi?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
export SMTP_ADDR='localhost:1025'   # необов'язково
go run ./cmd/api                    # міграції застосуються під час старту
```

### Команди `make`

| Команда | Дія |
|---|---|
| `make up` / `make down` | `docker compose up --build` / `docker compose down` |
| `make run` | запустити API на :8080 |
| `make run-engine` | запустити `pet-engine` |
| `make migrate` | застосувати міграції до `DATABASE_URL` |
| `make web` | запустити фронтенд на :5173 |
| `make build` | зібрати бінарні файли (`api`, `migrate`, `pet-engine`) у `bin/` |
| `make lint` | `gofmt` і `go vet` |
| `make generate` | перегенерувати код з контрактів: OpenAPI → Go-сервер і TS-клієнт, SQL → Go (sqlc у Docker) |
| `make deps-update` | оновити Go- і npm-залежності до останніх сумісних версій |

## Управління залежностями

| Частина | Маніфест і lock-файл | Версія середовища |
|---|---|---|
| Бекенд | [go.mod](go.mod) і [go.sum](go.sum): точні версії модулів і їхні контрольні суми | Go 1.26 (директива `go` у `go.mod`; її ж читає CI через `go-version-file`) |
| Інструменти бекенду | `oapi-codegen` закріплено директивою `tool` у `go.mod` і запускається через `go tool`, тож у всіх однакова версія без глобального встановлення. `sqlc` запускається з образу `sqlc/sqlc:1.31.1` (йому потрібен cgo) | — |
| Фронтенд | [web/package.json](web/package.json) і [web/package-lock.json](web/package-lock.json) | Node.js 22 (CI і `deploy/caddy.Dockerfile`) |
| Образи | `golang:1.26-alpine`, `node:22-alpine`, `caddy:2-alpine`, `postgres:17-alpine`, `redis:7-alpine`, `axllent/mailpit` | — |

Правила:

- Встановлення повторюване: `go mod download` і `npm ci` беруть точні версії з `go.sum` і `package-lock.json`. Так само працюють Docker-збірка та CI.
- Нова Go-залежність: `go get модуль@версія`, потім `go mod tidy`. Нова npm-залежність: `npm install пакет` у `web/`. Зміни `go.mod`/`go.sum` чи `package.json`/`package-lock.json` комітяться разом з кодом, що їх використовує.
- Оновлення: `make deps-update` (або точково `go get -u модуль`, `npm update пакет`), далі `go build ./... && go test ./...` і `npm run build`, і все це окремим pull request.
- Після зміни [api/openapi.yaml](api/openapi.yaml) або SQL-запитів виконайте `make generate` і закомітьте згенеровані файли (`internal/api/oapi/api.gen.go`, `web/src/api/schema.d.ts`, `internal/platform/pg/pgdb`).
- Секрети (`JWT_SECRET`, паролі БД і SMTP) передаються лише змінними середовища. `.env` у `.gitignore`, у репозиторії є тільки [.env.example](.env.example).

## Функціональні вимоги спринту 1

| Вимога | Стан | Де в коді | Як перевірити |
|---|---|---|---|
| FR-AUTH-01 Реєстрація за email і паролем | ✅ | [internal/auth/service.go](internal/auth/service.go) `Register`, `VerifyEmail`; [internal/auth/mail.go](internal/auth/mail.go) | `POST /auth/register`: пароль < 8 символів дає 400, дублікат email дає 409 `email already registered`. Лист з'являється в Mailpit; перехід за посиланням робить `emailVerified: true` у `GET /me`, а повторний перехід перенаправляє на `?verified=0` |
| FR-AUTH-03 Вхід, вихід, автоматичне оновлення сесії | ✅ | [internal/auth/handler.go](internal/auth/handler.go), [web/src/api/client.ts](web/src/api/client.ts) | Після входу в DevTools видно cookie `access_token`/`refresh_token` (httpOnly). Коли access-токен спливає, клієнт сам викликає `/auth/refresh`. `POST /auth/logout` очищає cookie |
| FR-PET-01 Створення улюбленця | ✅ | [internal/pets/pet.go](internal/pets/pet.go) `NewPet`, [internal/pets/service.go](internal/pets/service.go) `Create` | `POST /pets`: ім'я з 1 символу дає 400, другий улюбленець дає 409 |
| FR-PET-02 Показники 0–100 і стадія | ✅ | [internal/pets/stats.go](internal/pets/stats.go) `clamp`, [internal/pets/stage.go](internal/pets/stage.go) | `GET /pets/me` повертає `stats` (0–100) і `stage`: `egg` → `baby` (10 хв) → `child` (1 доба) → `teen` (3 доби) → `adult` (7 діб). У БД межі захищає `CHECK` |
| FR-PET-03 Дії догляду з перевіркою на сервері | ✅ | [internal/pets/service.go](internal/pets/service.go) `Perform`, [internal/pets/postgres.go](internal/pets/postgres.go) `ByOwnerForUpdate` | `POST /pets/me/actions {"action":"feed"}`: клієнт надсилає лише тип дії, сервер у транзакції з `FOR UPDATE` перераховує стан і повертає новий. Невідома дія дає 400 |
| FR-PET-04 Ліниве обчислення стану | ✅ | [internal/pets/pet.go](internal/pets/pet.go) `Recompute` | Після паузи `GET /pets/me` показує менші показники, хоча фонового процесу немає |
| FR-PET-05 Кулдауни | ✅ | [internal/pets/action.go](internal/pets/action.go) `DefaultRules`, `Pet.CooldownLeft` | Друге годування протягом 5 хв дає 429 із заголовком `Retry-After`; `cooldowns` у `GET /pets/me` |
| FR-PET-06 Інвентар і щоденна нагорода | ✅ | [internal/pets/inventory.go](internal/pets/inventory.go), [web/src/components/Backpack.tsx](web/src/components/Backpack.tsx) | `wash` списує мило; без мила відповідь 409. `POST /inventory/daily-reward` вдруге за добу (UTC) дає 429 з `Retry-After`; `dailyRewardIn` у `GET /inventory`. У UI це кнопка в наплічнику |
| FR-PET-07 Анімація за станом і дією | ✅ | [web/src/components/PetFigure.tsx](web/src/components/PetFigure.tsx), [web/src/styles.css](web/src/styles.css) | 4 стани (радіє, голодний, сонний, брудний) і анімація дії; з увімкненим `prefers-reduced-motion` анімація вимикається |
| FR-PET-09 Улюбленець не помирає | ✅ | [internal/pets/stats.go](internal/pets/stats.go), [internal/pets/pet.go](internal/pets/pet.go) | Після довгої відсутності показники опускаються до 0, але не нижче, і дії догляду їх відновлюють |

## Нефункціональні вимоги спринту 1

| Вимога | Як виконано | Чим підтверджено |
|---|---|---|
| NFR-SEC-01 HTTPS (TLS 1.2+) з HSTS | Caddy приймає лише TLS 1.2 і 1.3, HTTP перенаправляє 308 на HTTPS, додає `Strict-Transport-Security: max-age=31536000; includeSubDomains` | [deploy/Caddyfile](deploy/Caddyfile); `curl -I http://localhost:8080` дає 308 на https; `curl -skI https://localhost:8443` показує HSTS; `curl -sk --tls-max 1.1 https://localhost:8443` не встановлює з'єднання |
| NFR-SEC-02 Паролі лише як хеш | bcrypt; довжина пароля до 72 байт (обмеження bcrypt) | [internal/auth/adapters.go](internal/auth/adapters.go) `BcryptHasher`; `SELECT password_hash FROM users` показує хеш `$2a$…` |
| NFR-SEC-03 JWT у cookie; access ≤ 15 хв, refresh ≤ 30 днів з ротацією | Cookie `HttpOnly`, `Secure` (за HTTPS), `SameSite=Strict`; refresh-cookie лише для `/api/v1/auth`. Кожен refresh видає новий токен, а повторне використання старого відкликає всю сесію (Redis). `config.Load` не приймає TTL понад ліміти | [internal/auth/handler.go](internal/auth/handler.go), [internal/auth/refresh_redis.go](internal/auth/refresh_redis.go), [internal/config/config.go](internal/config/config.go); `document.cookie` у браузері не бачить токенів |
| NFR-SEC-06 Секрети в змінних середовища | Конфігурація лише з env; поза `APP_ENV=dev` API не стартує без `JWT_SECRET` | [.env.example](.env.example), [internal/config/config.go](internal/config/config.go); `.env` у `.gitignore` |
| NFR-USA-01 Адаптивний інтерфейс 360 px – десктоп | Mobile-first верстка на Tailwind | DevTools → режим пристрою 360 px і десктоп |
| NFR-MNT-01 OpenAPI як єдине джерело контракту | З [api/openapi.yaml](api/openapi.yaml) генеруються `oapi.ServerInterface` (Go) і TS-типи; запити перевіряються за специфікацією | `make generate-api`; нова операція в специфікації не скомпілюється, доки її не реалізовано в `app.Server`; невалідне тіло дає 400 з поясненням |
| NFR-MNT-02 Версійовані міграції goose, що застосовуються автоматично | Міграції вбудовані в бінарник і застосовуються під час старту API; є `cmd/migrate` | `docker compose up` на порожніх томах: у лозі `api` видно `migration applied` для кожної версії і `database schema is up to date` |
| NFR-MNT-04 CI: лінтер, тести, збірка на кожен PR | GitHub Actions: `gofmt`, `go vet`, `go test -race`, збірка бекенду, typecheck і збірка фронтенду | [.github/workflows/ci.yml](.github/workflows/ci.yml); branch protection на `main` |
| NFR-MNT-06 Локальне середовище однією командою | Compose: `api`, `postgres`, `redis`, `caddy`, `mailpit` з healthcheck-ами й залежностями | `docker compose up --build` після клонування |

## Команда та ролі

| Учасник | Ролі | Зона відповідальності в коді |
|---|---|---|
| Волошенко Михайло Олександрович | Scrum-майстер, бекенд-розробник, ШІ-розробник | `internal/auth`, `internal/pets`, `internal/brain`, `internal/platform`, `internal/app`; далі — `social`, `trade`, `leaderboard`, `notify`, `replica` |
| Березняцький Олександр Ігорович | Фронтенд-розробник, DevOps-інженер | `web/`, `cmd/`, `Makefile`, `.env.example`; далі — Docker Compose, CI, Caddy, деплой |
| Кертичак Павло Сергійович | Аналітик (Product Owner), тестувальник (QA) | `api/openapi.yaml`, критерії приймання, рецензування pull request'ів; далі — автотести (`tests/`) |

## Робота з Git

- `main` — стабільна гілка; після початкового коміту прямі коміти в неї заборонені (branch protection).
- Кожна задача з Jira (проєкт `SCRUM`) виконується в окремій гілці: `feature/SCRUM-<номер>-<коротка-назва>`, наприклад `feature/SCRUM-32-lazy-recompute`, `feature/SCRUM-26-web-skeleton`.
- Зміни потрапляють у `main` лише через pull request з рецензією щонайменше одного іншого учасника.
- Повідомлення комітів — у наказовому способі з ключем задачі: `SCRUM-32: add lazy pet state recomputation`.

## Ліцензія

[MIT](LICENSE)