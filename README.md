# Тамагочі

Веб-гра з віртуальним улюбленцем: догляд з браузера або з головного екрана телефона (PWA), характер улюбленця на основі utility AI, друзі, рейтинг, обмін предметами та репліки улюбленця через LLM.

Проєкт виконується в межах дисципліни «Життєвий цикл програмного забезпечення» (КПІ ім. Ігоря Сікорського, ФІОТ).

> **Стан:** базова структура коду (лабораторна робота №4). Повністю працює ядро спринту 1: реєстрація, вхід, улюбленець і дії догляду з лінивим перерахунком стану. Решта модулів має сутності, інтерфейси (порти) та сигнатури сервісів; їхні методи поки повертають `apperr.ErrNotImplemented` і реалізуються в наступних спринтах.

## Стек

| Рівень | Технології |
|---|---|
| Бекенд | Go 1.24, chi, JWT (golang-jwt), bcrypt |
| Фронтенд | React 18, Vite, TypeScript, TanStack Query |
| Заплановано | PostgreSQL (pgx, sqlc, goose), Redis, asynq, WebSocket, Web Push, Docker Compose, Caddy |

## Архітектура

Модульний моноліт на Go плюс окремий процес `pet-engine`, що збирається з того самого репозиторію (документ «Архітектура програмного забезпечення», лабораторна робота №2).

Кожен модуль — окремий Go-пакет із доменними типами, портами (інтерфейсами) та сервісом. Модулі не звертаються до сховищ або сервісів напряму, а отримують залежності через інтерфейси. Конкретні реалізації створюються лише в composition root [internal/app](internal/app/app.go).

## Структура проєкту

```
tamagochi/
├── cmd/
│   ├── api/                 # точка входу HTTP-сервера
│   └── pet-engine/          # точка входу фонового воркера (LLM, push, обміни)
├── internal/
│   ├── app/                 # composition root: зв'язування модулів
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
│       └── tx/              #   TxManager (межа транзакції)
├── api/openapi.yaml         # контракт API (NFR-MNT-01)
├── web/                     # фронтенд (React + Vite + TS)
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
| auth | `UserRepository`, `PasswordHasher`, `TokenIssuer`, `Mailer` | порти | ✅ + in-memory, bcrypt, JWT адаптери |
| pets | `Pet` | `Recompute` (ліниве обчислення), `Apply`, `CooldownLeft` | ✅ |
| pets | `ActionRule` / `StatRule` | правила дій (кулдаун, ефект, предмет, очки) | ✅ |
| pets | `Inventory`, `Item`, `Catalog` | інвентар і каталог предметів | ✅ |
| pets | `Service` | `Create`, `Get`, `Perform` у транзакції | ✅ |
| pets | `PetRepository`, `InventoryRepository`, `Decider` | порти | ✅ + in-memory адаптери |
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
- **L — підстановка Лісков.** Будь-яка реалізація порту (`MemoryPetRepository`, а пізніше PostgreSQL-адаптер) підставляється в `pets.Service` без зміни його поведінки.
- **I — розділення інтерфейсів.** Порти вузькі: `Decider` має один метод, `TokenVerifier` у middleware — лише перевірку токена. `trade` бачить інвентар лише через `InventoryPort`, а не весь сервіс `pets`.
- **D — інверсія залежностей.** Сервіси залежать від абстракцій (`PetRepository`, `tx.Manager`, `clock.Clock`, `LLMProvider`). Конкретні адаптери підставляються в [internal/app](internal/app/app.go).

## Запуск

Потрібні Go 1.24+ і Node.js 20+.

```bash
# бекенд на http://localhost:8080
cp .env.example .env      # необов'язково: у режимі dev є значення за замовчуванням
go run ./cmd/api

# фронтенд на http://localhost:5173 (проксіює /api на :8080)
cd web
npm install
npm run dev
```

Те саме через `make`:

| Команда | Дія |
|---|---|
| `make run` | запустити API на :8080 |
| `make run-engine` | запустити `pet-engine` |
| `make web` | запустити фронтенд на :5173 |
| `make build` | зібрати бінарні файли в `bin/` |
| `make lint` | `gofmt` і `go vet` |
| `make up` | `docker compose up --build` |
| `make down` | `docker compose down` |

### Запуск однією командою (Docker Compose)

Потрібен лише Docker з Compose v2. Після клонування репозиторію:

```bash
docker compose up --build
```

Підіймаються чотири сервіси: `api`, `postgres`, `redis` і `caddy`. Застосунок доступний на http://localhost:8080 (порт змінюється через `CADDY_HTTP_PORT`). Caddy віддає зібраний фронтенд і проксіює `/api` та `/healthz` на `api`. Усі змінні мають значення за замовчуванням, тож файл `.env` для першого запуску не потрібен; за потреби скопіюйте `.env.example` у `.env`.

```bash
docker compose down        # зупинити
docker compose down -v     # зупинити й видалити дані postgres/redis
```

Приклад запитів:

```bash
curl -X POST localhost:8080/api/v1/auth/register -d '{"email":"a@b.c","password":"secret123","nickname":"misha"}'
curl -c jar -X POST localhost:8080/api/v1/auth/login -d '{"email":"a@b.c","password":"secret123"}'
curl -b jar -X POST localhost:8080/api/v1/pets -d '{"name":"Bublyk","appearance":"cat"}'
curl -b jar -X POST localhost:8080/api/v1/pets/me/actions -d '{"action":"feed"}'
```

Дані зберігаються в пам'яті процесу й зникають після перезапуску. Адаптери PostgreSQL і Redis підключаються в наступних спринтах.

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