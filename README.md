# Koteyye Music Backend

Бэкенд для музыкального стримингового сервиса, написанный на Go с использованием Clean Architecture.

## Стек технологий

- **Язык**: Go 1.21+
- **Web Framework**: Chi
- **База данных**: PostgreSQL (pgx)
- **Object Storage**: MinIO (S3-compatible)
- **Аутентификация**: JWT (golang-jwt/jwt/v5)
- **Обработка аудио**: FFmpeg
- **Хеширование паролей**: bcrypt

## Особенности

- Регистрация и авторизация пользователей
- OAuth авторизация через Google и Yandex
- Система прав доступа на основе ролей (RBAC)
- Загрузка треков с автоматической конвертацией в MP3 (320kbps)
- Стриминг аудио с поддержкой Range Requests (перемотка)
- Пагинация списка треков
- Получение треков пользователя
- Админские функции (загрузка и удаление треков)
- Поддержка загрузки обложек альбомов
- CORS для работы с фронтендом

## Структура проекта

```
koteyye_music_be/
├── cmd/
│   └── api/
│       └── main.go              # Точка входа приложения
├── internal/
│   ├── config/
│   │   └── config.go           # Конфигурация приложения
│   ├── handler/
│   │   ├── auth_handler.go     # HTTP обработчики авторизации
│   │   └── track_handler.go    # HTTP обработчики треков
│   ├── middleware/
│   │   ├── auth.go             # JWT middleware
│   │   └── cors.go             # CORS middleware
│   ├── models/
│   │   ├── user.go             # Модели пользователя
│   │   └── track.go            # Модели трека
│   ├── repository/
│   │   ├── user_repository.go  # Репозиторий пользователя
│   │   └── track_repository.go # Репозиторий трека
│   └── service/
│       ├── auth_service.go     # Сервис авторизации
│       └── track_service.go    # Сервис треков
├── pkg/
│   ├── database/
│   │   └── database.go         # Подключение к БД
│   ├── logger/
│   │   └── logger.go           # Логирование
│   └── minio/
│       └── minio.go            # MinIO клиент
├── migrations/
│   └── init.sql                # Инициализация БД
├── scripts/
│   └── docker-compose.yml      # PostgreSQL и MinIO
├── .env.example                # Пример переменных окружения
├── go.mod                      # Зависимости Go
└── README.md                   # Документация
```

## Установка и запуск

### Требования

- Go 1.21+
- Docker и Docker Compose
- FFmpeg (для конвертации аудио)

### 1. Клонирование репозитория

```bash
git clone https://gitflic.ru/project/koteyye/koteyye_music_be.git
cd koteyye_music_be
```

### 2. Установка зависимостей

```bash
go mod download
```

### 3. Настройка PostgreSQL и MinIO

Запустите контейнеры с базой данных и хранилищем:

```bash
cd scripts
docker-compose up -d
```

### 4. Создание базы данных

```bash
# Создайте пустую базу данных (если еще не создана)
createdb music_service
```

**Важно:** Миграции выполняются **автоматически** при запуске приложения. Вам не нужно выполнять их вручную.

Автоматическая система миграций:
- ✅ Запускается при старте приложения
- ✅ Применяет только новые миграции
- ✅ Отслеживает примененные миграции в таблице `schema_migrations`
- ✅ Выполняет миграции в правильном порядке (001, 002, 003...)
- ✅ Поддерживает транзакции (откат при ошибке)

### 5. Настройка переменных окружения

Скопируйте `.env.example` в `.env` и отредактируйте при необходимости:

```bash
cp .env.example .env
```

Пример содержимого `.env`:

```env
DB_DSN=postgres://postgres_user:postgres_pass@localhost:5432/music_service?sslmode=disable
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=music-files
MINIO_USE_SSL=false
JWT_SECRET=your-super-secret-jwt-key-change-in-production
SERVER_PORT=8080
```

### 6. Запуск приложения

```bash
go run cmd/api/main.go
```

Сервер запустится на `http://localhost:8080`

## API Эндпоинты

### Авторизация

- `POST /auth/register` - Регистрация нового пользователя
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```

- `POST /auth/login` - Вход в систему
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```

- `GET /auth/google/login` - Начало авторизации через Google (редирект на Google)
- `GET /auth/google/callback` - Callback от Google (редирект на фронтенд с токеном)
- `GET /auth/yandex/login` - Начало авторизации через Yandex (редирект на Yandex)
- `GET /auth/yandex/callback` - Callback от Yandex (редирект на фронтенд с токеном)

### Треки (требуется авторизация)

- `POST /api/tracks/upload` - Загрузка трека (multipart/form-data)
  - `title`: название трека
  - `artist`: исполнитель (опционально)
  - `album`: альбом (опционально)
  - `audio`: аудиофайл (mp3, wav, flac)
  - `image`: обложка (опционально)

- `GET /api/tracks` - Список треков с пагинацией
  - `page`: номер страницы (по умолчанию 1)
  - `limit`: количество на странице (по умолчанию 20, максимум 100)

- `GET /api/tracks/my` - Треки текущего пользователя

- `GET /api/tracks/{id}/stream` - Стриминг трека с поддержкой перемотки

- `DELETE /api/tracks/{id}` - Удаление трека

### Другое

- `GET /health` - Проверка здоровья сервиса
- `GET /api/docs` - Swagger UI (интерактивная документация API)
- `GET /api/openapi.yaml` - OpenAPI спецификация (YAML)

## Установка FFmpeg

### macOS (Homebrew)

```bash
brew install ffmpeg
```

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install ffmpeg
```

### Windows

Скачайте с https://ffmpeg.org/download.html и добавьте в PATH

## API Документация

### Swagger UI

Приложение включает в себя интерактивную документацию API на основе Swagger UI:

- **URL**: http://localhost:8080/api/docs
- **Функции**:
  - Просмотр всех API эндпоинтов
  - Интерактивное тестирование запросов (Try it out)
  - Автоматическая генерация запросов и ответов
  - Скачивание OpenAPI спецификации
  - Подсветка синтаксиса
  - Поиск по эндпоинтам

### OpenAPI Спецификация

Полная спецификация API в формате OpenAPI 3.0.3 доступна по адресу:
- **URL**: http://localhost:8080/api/openapi.yaml
- **Формат**: YAML
- **Версия**: 2.0.0

### Использование в клиентских приложениях

Вы можете использовать OpenAPI спецификацию для:
- Генерации клиентских библиотек (OpenAPI Generator)
- Автоматического создания типов (TypeScript, Java, Python, etc.)
- Валидации запросов и ответов
- Создания документации

Пример генерации клиента:
```bash
# Установка OpenAPI Generator
npm install -g @openapitools/openapi-generator-cli

# Генерация TypeScript клиента
openapi-generator-cli generate \
  -i http://localhost:8080/api/openapi.yaml \
  -g typescript-axios \
  -o ./generated-client
```

## Конфигурация

Все настройки загружаются из переменных окружения:

| Переменная | Описание | Значение по умолчанию |
|------------|----------|----------------------|
| DB_DSN | DSN для подключения к PostgreSQL | postgres://postgres_user:postgres_pass@localhost:5432/music_service?sslmode=disable |
| MINIO_ENDPOINT | Адрес MinIO сервера | localhost:9000 |
| MINIO_ACCESS_KEY | Access key для MinIO | minioadmin |
| MINIO_SECRET_KEY | Secret key для MinIO | minioadmin |
| MINIO_BUCKET | Имя бакета | music-files |
| MINIO_USE_SSL | Использовать SSL для MinIO | false |
| JWT_SECRET | Секретный ключ для JWT | default-secret-key-change-in-production |
| SERVER_PORT | Порт сервера | 8080 |
| GOOGLE_CLIENT_ID | Client ID для Google OAuth | - |
| GOOGLE_CLIENT_SECRET | Client Secret для Google OAuth | - |
| GOOGLE_REDIRECT_URL | Redirect URL для Google OAuth | http://localhost:8080/auth/google/callback |
| YANDEX_CLIENT_ID | Client ID для Yandex OAuth | - |
| YANDEX_CLIENT_SECRET | Client Secret для Yandex OAuth | - |
| YANDEX_REDIRECT_URL | Redirect URL для Yandex OAuth | http://localhost:8080/auth/yandex/callback |
| FRONTEND_URL | URL фронтенда для редиректа после OAuth | http://localhost:5173 |

## Архитектура

Проект использует Clean Architecture (слоистую архитектуру):

1. **Handler Layer** - Обработка HTTP запросов и ответов
2. **Service Layer** - Бизнес-логика (обработка аудио, работа с MinIO)
3. **Repository Layer** - Доступ к данным (SQL запросы к PostgreSQL)
4. **Models** - Структуры данных
5. **Middleware** - Перехватчики запросов (авторизация, CORS, логирование, проверка прав)

## OAuth Настройка

Для настройки OAuth авторизации через Google и Yandex следуйте инструкции в файле [OAUTH_SETUP.md](OAUTH_SETUP.md).

Краткий обзор:

1. **Google OAuth**:
   - Создайте проект в [Google Cloud Console](https://console.cloud.google.com/)
   - Настройте OAuth Consent Screen
   - Создайте OAuth Client ID типа Web Application
   - Добавьте `http://localhost:8080/auth/google/callback` в Redirect URIs
   - Получите Client ID и Client Secret

2. **Yandex OAuth**:
   - Создайте приложение на [Яндекс OAuth](https://oauth.yandex.ru/)
   - Установите права: Доступ к логину и электронной почте
   - Добавьте `http://localhost:8080/auth/yandex/callback` в Redirect URI
   - Получите Client ID и Client Secret
### Автоматические миграции

**Миграции выполняются автоматически при запуске приложения!**

При первом запуске приложение:
1. Создаст все необходимые таблицы
2. Применит миграции OAuth (если нужно)
3. Применит миграции RBAC (если нужно)
4. Создаст таблицу `schema_migrations` для отслеживания

Ничего выполнять вручную не нужно!

Для проверки статуса миграций:
- Приложение логирует каждую примененную миграцию
- Посмотрите логи при запуске для статуса

Структура миграций:
```
migrations/
├── 001_init.sql        # Инициализация БД (users, tracks)
├── 002_add_oauth.sql   # Поля OAuth (provider, external_id)
└── 003_add_role.sql   # RBAC поле role
```

4. **Настройка .env**:
   ```env
   GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your-google-client-secret
   YANDEX_CLIENT_ID=your-yandex-client-id
   YANDEX_CLIENT_SECRET=your-yandex-client-secret
   FRONTEND_URL=http://localhost:5173
   ```

## Тестирование

Примеры запросов:

### Регистрация пользователя

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

### Вход в систему

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

Сохраните полученный токен для дальнейших запросов.

### Загрузка трека

```bash
curl -X POST http://localhost:8080/api/tracks/upload \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -F "title=My Track" \
  -F "artist=Artist Name" \
  -F "audio=@/path/to/your/song.mp3" \
  -F "image=@/path/to/your/cover.jpg"
```

### Стриминг трека

```bash
curl -X GET http://localhost:8080/api/tracks/TRACK_ID/stream \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  --output output.mp3
```

### Получение списка треков

```bash
curl -X GET "http://localhost:8080/api/tracks?page=1&limit=10" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## RBAC и Администрирование

Для настройки системы прав доступа и администрирования следуйте инструкции в файле [ADMIN_GUIDE.md](ADMIN_GUIDE.md).

Краткий обзор:

1. **Роли пользователей**:
   - `user` - обычный пользователь (прослушивание треков)
   - `admin` - администратор (загрузка и удаление треков)

2. **Миграции БД**:
   ### Автоматические миграции

   **Миграции выполняются автоматически при запуске приложения!**

   При первом запуске приложение автоматически:
   1. Создаст таблицу `schema_migrations`
   2. Применит все необходимые миграции
   3. Отобразит статус в логах

   Пример логов при запуске:
   ```
   INFO Database connected successfully
   INFO Running database migrations...
   INFO Applying migration file=001_init.sql
   INFO Migration applied successfully file=001_init.sql
   INFO Applying migration file=002_add_oauth.sql
   INFO Migration applied successfully file=002_add_oauth.sql
   INFO Applying migration file=003_add_role.sql
   INFO Migration applied successfully file=003_add_role.sql
   INFO Migrations completed applied=3 total=3
   INFO Server started addr=:8080
   ```

   Ничего выполнять вручную не нужно!

3. **Назначение администратора**:
   ```sql
   UPDATE users SET role = 'admin' WHERE email = 'admin@example.com';
   ```

4. **JWT токен содержит роль**:
   ```json
   {
     "user_id": 1,
     "email": "admin@example.com",
     "role": "admin",
     ...
   }
   ```

5. **Админские роуты** (требуют роль 'admin'):
   - `POST /api/admin/tracks/upload` - Загрузка трека
   - `DELETE /api/admin/tracks/{id}` - Удаление трека

## Разработка

### Тестирование OAuth

После настройки OAuth:

```bash
# Google OAuth (откроется браузер)
curl -L http://localhost:8080/auth/google/login

# Yandex OAuth (откроется браузер)
curl -L http://localhost:8080/auth/yandex/login
```

После успешной авторизации вы будете перенаправлены на Frontend с токеном:
```
http://localhost:5173/auth-callback?token=JWT_TOKEN&provider=google
```

### Добавление новых зависимостей

```bash
go get github.com/example/package
go mod tidy
```

### Запуск тестов

```bash
go test ./...
```

### Линтинг

```bash
go vet ./...
go fmt ./...
```

## Вклад в проект

Для внесения вклада в проект:

1. Форкните репозиторий
2. Создайте ветку для вашей фичи (`git checkout -b feature/AmazingFeature`)
3. Закоммитьте изменения (`git commit -m 'Add some AmazingFeature'`)
4. Запушьте в ветку (`git push origin feature/AmazingFeature`)
5. Откройте Pull Request

## Лицензия

Этот проект распространяется под лицензией MIT.

## Контакты

Для вопросов и предложений открывайте Issue в репозитории.