# API Документация

Полная документация REST API музыкального стримингового сервиса Koteyye Music.

## Содержание

1. [Обзор](#обзор)
2. [Аутентификация](#аутентификация)
3. [Базовый URL](#базовый-url)
4. [Эндпоинты](#эндпоинты)
5. [Модели данных](#модели-данных)
6. [Коды ответов](#коды-ответов)
7. [Примеры использования](#примеры-использования)
8. [Инструменты](#инструменты)

---

## Обзор

### Особенности API

- ✅ RESTful архитектура
- ✅ JWT авторизация с ролями (user/admin)
- ✅ OAuth 2.0 (Google, Yandex)
- ✅ Загрузка треков с автоматической конвертацией
- ✅ Стриминг с поддержкой Range Requests
- ✅ Пагинация списков
- ✅ CORS поддержка
- ✅ OpenAPI 3.0 спецификация
- ✅ Интерактивная Swagger UI документация

### Технические детали

- **Формат**: JSON (за исключением бинарных данных)
- **Кодировка**: UTF-8
- **Версия API**: 2.0.0
- **Версия протокола**: HTTP/1.1 или HTTP/2

---

## Аутентификация

### JWT (Bearer Token)

Для доступа к защищенным эндпоинтам требуется JWT токен в заголовке Authorization:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### Получение токена

Есть три способа получения JWT токена:

1. **Регистрация** - создает нового пользователя
2. **Логин** - авторизация по email и паролю
3. **OAuth** - авторизация через Google или Yandex

#### Структура JWT токена

```json
{
  "user_id": 1,
  "email": "user@example.com",
  "role": "user",
  "exp": 1705335600,
  "iat": 1705249200,
  "nbf": 1705249200
}
```

#### Срок действия

- **Время жизни**: 24 часа
- **Обновление**: требуется повторная авторизация

### Роли пользователей

| Роль | Описание | Доступные эндпоинты |
|------|----------|---------------------|
| `user` | Обычный пользователь | GET /api/tracks<br>GET /api/tracks/my<br>GET /api/tracks/{id}/stream |
| `admin` | Администратор | Все эндпоинты пользователя<br>POST /api/admin/tracks/upload<br>DELETE /api/admin/tracks/{id} |

---

## Базовый URL

### Локальная разработка

```
http://localhost:8080
```

### Production

```
https://api.koteyye-music.com
```

---

## Эндпоинты

### Health Check

#### Проверка состояния сервиса

**GET** `/health`

Проверяет работоспособность сервиса.

**Ответы:**

- `200 OK` - Сервис работает нормально

**Пример ответа:**

```http
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Content-Length: 2

OK
```

---

### Аутентификация

#### Регистрация пользователя

**POST** `/auth/register`

Регистрирует нового пользователя в системе.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Валидация:**

- `email` (required, string, email format)
- `password` (required, string, min 6 characters)

**Ответы:**

- `201 Created` - Пользователь успешно зарегистрирован
- `409 Conflict` - Пользователь с таким email уже существует
- `400 Bad Request` - Некорректные данные

**Пример успешного ответа (201):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "role": "user",
    "provider": "local",
    "external_id": "",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

**Пример ошибки (409):**

```json
{
  "error": "user with this email already exists"
}
```

**Пример ошибки (400):**

```json
{
  "error": "Email is required"
}
```

#### Вход в систему

**POST** `/auth/login`

Авторизует пользователя по email и паролю.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Валидация:**

- `email` (required, string, email format)
- `password` (required, string)

**Ответы:**

- `200 OK` - Успешная авторизация
- `401 Unauthorized` - Неверный email или пароль
- `400 Bad Request` - Некорректные данные

**Пример успешного ответа (200):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "role": "admin",
    "provider": "local",
    "external_id": "",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

**Пример ошибки (401):**

```json
{
  "error": "invalid email or password"
}
```

---

### OAuth 2.0

#### Google OAuth - Начало

**GET** `/auth/google/login`

Инициирует процесс OAuth авторизации через Google.

Перенаправляет пользователя на страницу авторизации Google.

**Ответы:**

- `307 Temporary Redirect` - Перенаправление на Google

**Пример:**

```http
HTTP/1.1 307 Temporary Redirect
Location: https://accounts.google.com/o/oauth2/v2/auth?client_id=...&redirect_uri=...&response_type=code&scope=...
```

#### Google OAuth - Callback

**GET** `/auth/google/callback`

Обрабатывает ответ от Google и перенаправляет на фронтенд с JWT токеном.

**Query Parameters:**

- `code` (required, string) - Код авторизации от Google
- `state` (optional, string) - Параметр состояния для CSRF защиты

**Ответы:**

- `307 Temporary Redirect` - Перенаправление на фронтенд с токеном
- `500 Internal Server Error` - Ошибка обработки OAuth

**Пример перенаправления:**

```http
HTTP/1.1 307 Temporary Redirect
Location: http://localhost:5173/auth-callback?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...&provider=google
```

#### Yandex OAuth - Начало

**GET** `/auth/yandex/login`

Инициирует процесс OAuth авторизации через Yandex.

**Ответы:**

- `307 Temporary Redirect` - Перенаправление на Yandex

#### Yandex OAuth - Callback

**GET** `/auth/yandex/callback`

Обрабатывает ответ от Yandex и перенаправляет на фронтенд с JWT токеном.

**Query Parameters:**

- `code` (required, string) - Код авторизации от Yandex
- `state` (optional, string) - Параметр состояния для CSRF защиты

**Ответы:**

- `307 Temporary Redirect` - Перенаправление на фронтенд с токеном
- `500 Internal Server Error` - Ошибка обработки OAuth

---

### Треки (Общедоступные)

#### Список треков

**GET** `/api/tracks`

Возвращает пагинированный список всех треков.

**Authorization:** Bearer Token (required)

**Query Parameters:**

- `page` (optional, integer, default: 1, min: 1) - Номер страницы
- `limit` (optional, integer, default: 20, min: 1, max: 100) - Количество треков на странице

**Пример запроса:**

```http
GET /api/tracks?page=1&limit=20 HTTP/1.1
Host: localhost:8080
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Ответы:**

- `200 OK` - Список треков получен
- `401 Unauthorized` - Требуется авторизация

**Пример ответа (200):**

```json
{
  "tracks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": 1,
      "title": "My Awesome Track",
      "artist": "Artist Name",
      "album": "Album Name",
      "duration_seconds": 180,
      "s3_audio_key": "audio/550e8400-e29b-41d4-a716-446655440000.mp3",
      "s3_image_key": "images/550e8400-e29b-41d4-a716-446655440000.jpg",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
```

#### Треки пользователя

**GET** `/api/tracks/my`

Возвращает все треки, загруженные текущим пользователем.

**Authorization:** Bearer Token (required)

**Ответы:**

- `200 OK` - Список треков получен
- `401 Unauthorized` - Требуется авторизация

**Пример ответа (200):**

```json
{
  "tracks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": 1,
      "title": "My Track",
      "artist": "Artist Name",
      "album": "Album Name",
      "duration_seconds": 180,
      "s3_audio_key": "audio/550e8400-e29b-41d4-a716-446655440000.mp3",
      "s3_image_key": "images/550e8400-e29b-41d4-a716-446655440000.jpg",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### Стриминг трека

**GET** `/api/tracks/{id}/stream`

Стримит аудиофайл с поддержкой Range Requests (перемотка).

**Authorization:** Bearer Token (required)

**Path Parameters:**

- `id` (required, string, format: uuid) - UUID трека

**Ответы:**

- `200 OK` - Полный файл (начало воспроизведения)
- `206 Partial Content` - Частичный файл (перемотка)
- `401 Unauthorized` - Требуется авторизация
- `404 Not Found` - Трек не найден

**Пример ответа (200):**

```http
HTTP/1.1 200 OK
Content-Type: audio/mpeg
Accept-Ranges: bytes
Content-Length: 5242880
Last-Modified: Mon, 15 Jan 2024 10:30:00 GMT

[binary audio data]
```

**Пример ответа (206):**

```http
HTTP/1.1 206 Partial Content
Content-Type: audio/mpeg
Accept-Ranges: bytes
Content-Length: 1024
Content-Range: bytes 1024-2047/5242880
Last-Modified: Mon, 15 Jan 2024 10:30:00 GMT

[binary audio data]
```

---

### Администраторские функции

#### Загрузка трека

**POST** `/api/admin/tracks/upload`

Загружает новый трек в систему с автоматической конвертацией в MP3 (320kbps).

**Authorization:** Bearer Token + Admin Role (required)

**Request Body:** multipart/form-data

**Form Fields:**

- `title` (required, string) - Название трека
- `artist` (optional, string) - Исполнитель
- `album` (optional, string) - Альбом
- `audio` (required, file) - Аудиофайл (mp3, wav, flac)
- `image` (optional, file) - Обложка (jpg, png, gif, webp)

**Content-Type:** `multipart/form-data`

**Максимальный размер файла:** 100MB

**Ответы:**

- `201 Created` - Трек успешно загружен
- `401 Unauthorized` - Требуется авторизация
- `403 Forbidden` - Недостаточно прав доступа (требуется роль admin)
- `400 Bad Request` - Некорректные данные
- `500 Internal Server Error` - Ошибка при обработке файла

**Пример запроса (curl):**

```bash
curl -X POST http://localhost:8080/api/admin/tracks/upload \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN" \
  -F "title=My New Track" \
  -F "artist=Artist Name" \
  -F "album=Album Name" \
  -F "audio=@/path/to/track.mp3" \
  -F "image=@/path/to/cover.jpg"
```

**Пример запроса (JavaScript/FormData):**

```javascript
const formData = new FormData();
formData.append('title', 'My New Track');
formData.append('artist', 'Artist Name');
formData.append('album', 'Album Name');
formData.append('audio', audioFile);
formData.append('image', imageFile);

fetch('http://localhost:8080/api/admin/tracks/upload', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`
  },
  body: formData
})
  .then(response => response.json())
  .then(data => console.log(data));
```

**Пример ответа (201):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 1,
  "title": "My New Track",
  "artist": "Artist Name",
  "album": "Album Name",
  "duration_seconds": 180,
  "s3_audio_key": "audio/550e8400-e29b-41d4-a716-446655440000.mp3",
  "s3_image_key": "images/550e8400-e29b-41d4-a716-446655440000.jpg",
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### Удаление трека

**DELETE** `/api/admin/tracks/{id}`

Удаляет трек из системы, включая файлы из MinIO и запись в БД.

**Authorization:** Bearer Token + Admin Role (required)

**Path Parameters:**

- `id` (required, string, format: uuid) - UUID трека

**Логика удаления:**

1. Получает данные трека из БД
2. Удаляет аудиофайл из MinIO
3. Удаляет обложку из MinIO (если есть)
4. Удаляет запись из БД
5. Все операции консистентны

**Ответы:**

- `204 No Content` - Трек успешно удален
- `401 Unauthorized` - Требуется авторизация
- `403 Forbidden` - Недостаточно прав доступа (требуется роль admin)
- `404 Not Found` - Трек не найден
- `500 Internal Server Error` - Ошибка при удалении

**Пример запроса:**

```bash
curl -X DELETE http://localhost:8080/api/admin/tracks/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN"
```

**Пример ответа (204):**

```http
HTTP/1.1 204 No Content
```

---

## Модели данных

### User

Пользователь системы.

```json
{
  "id": 1,
  "email": "user@example.com",
  "role": "user",
  "provider": "local",
  "external_id": "",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Поля:**

| Поле | Тип | Описание |
|------|------|----------|
| `id` | integer | Уникальный идентификатор пользователя |
| `email` | string (email) | Email пользователя |
| `role` | enum (`user`, `admin`) | Роль пользователя |
| `provider` | enum (`local`, `google`, `yandex`) | Способ авторизации |
| `external_id` | string (nullable) | ID пользователя у OAuth провайдера |
| `created_at` | datetime | Дата и время создания аккаунта |

### Track

Трек музыкальной библиотеки.

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": 1,
  "title": "My Awesome Track",
  "artist": "Artist Name",
  "album": "Album Name",
  "duration_seconds": 180,
  "s3_audio_key": "audio/550e8400-e29b-41d4-a716-446655440000.mp3",
  "s3_image_key": "images/550e8400-e29b-41d4-a716-446655440000.jpg",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Поля:**

| Поле | Тип | Описание |
|------|------|----------|
| `id` | string (uuid) | Уникальный идентификатор трека |
| `user_id` | integer | ID пользователя, загрузившего трек |
| `title` | string | Название трека |
| `artist` | string (nullable) | Исполнитель |
| `album` | string (nullable) | Альбом |
| `duration_seconds` | integer | Длительность трека в секундах |
| `s3_audio_key` | string | Ключ аудиофайла в S3 |
| `s3_image_key` | string (nullable) | Ключ обложки в S3 |
| `created_at` | datetime | Дата и время загрузки трека |

### Error

Стандартный формат ошибок.

```json
{
  "error": "Error message"
}
```

**Поля:**

| Поле | Тип | Описание |
|------|------|----------|
| `error` | string | Сообщение об ошибке |

### Pagination

Информация о пагинации списков.

```json
{
  "page": 1,
  "limit": 20,
  "total": 100
}
```

**Поля:**

| Поле | Тип | Описание |
|------|------|----------|
| `page` | integer | Текущая страница |
| `limit` | integer | Количество элементов на странице |
| `total` | integer | Общее количество элементов |

---

## Коды ответов

### 2xx Success

| Код | Описание |
|------|----------|
| 200 OK | Запрос успешно выполнен |
| 201 Created | Ресурс успешно создан |
| 204 No Content | Запрос успешно выполнен, без содержимого |
| 206 Partial Content | Часть контента (для Range Requests) |

### 3xx Redirection

| Код | Описание |
|------|----------|
| 307 Temporary Redirect | Временное перенаправление (OAuth) |

### 4xx Client Error

| Код | Описание |
|------|----------|
| 400 Bad Request | Некорректные данные запроса |
| 401 Unauthorized | Требуется авторизация или невалидный токен |
| 403 Forbidden | Недостаточно прав доступа |
| 404 Not Found | Ресурс не найден |
| 409 Conflict | Конфликт данных (например, email уже существует) |

### 5xx Server Error

| Код | Описание |
|------|----------|
| 500 Internal Server Error | Внутренняя ошибка сервера |

---

## Примеры использования

### Регистрация и получение токена

```bash
# Шаг 1: Регистрация
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'

# Ответ:
# {
#   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "user": {
#     "id": 1,
#     "email": "user@example.com",
#     "role": "user",
#     "created_at": "2024-01-15T10:30:00Z"
#   }
# }

# Шаг 2: Сохраните токен
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Загрузка трека (администратор)

```bash
# Предварительно назначьте роль admin в БД
# UPDATE users SET role = 'admin' WHERE email = 'user@example.com';

# Войдите для получения admin токена
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'

# Сохраните ADMIN_TOKEN

# Загрузите трек
curl -X POST http://localhost:8080/api/admin/tracks/upload \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "title=My Track" \
  -F "artist=Artist Name" \
  -F "audio=@/path/to/song.mp3" \
  -F "image=@/path/to/cover.jpg"

# Ответ:
# {
#   "id": "550e8400-e29b-41d4-a716-446655440000",
#   "title": "My Track",
#   "artist": "Artist Name",
#   ...
# }
```

### Получение списка треков с пагинацией

```bash
curl -X GET "http://localhost:8080/api/tracks?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"

# Ответ:
# {
#   "tracks": [...],
#   "pagination": {
#     "page": 1,
#     "limit": 10,
#     "total": 100
#   }
# }
```

### Стриминг трека с перемоткой

```bash
# Начало воспроизведения
curl -X GET "http://localhost:8080/api/tracks/550e8400-e29b-41d4-a716-446655440000/stream" \
  -H "Authorization: Bearer $TOKEN" \
  --output track.mp3

# Перемотка на 5 минут (300 секунд)
curl -X GET "http://localhost:8080/api/tracks/550e8400-e29b-41d4-a716-446655440000/stream" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Range: bytes=2560000-" \
  --output track_part.mp3
```

### Удаление трека (администратор)

```bash
curl -X DELETE "http://localhost:8080/api/admin/tracks/550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Ответ: 204 No Content
```

---

## Инструменты

### Swagger UI

Интерактивная документация API с возможностью тестирования запросов.

- **URL**: http://localhost:8080/api/docs
- **Функции**:
  - Просмотр всех эндпоинтов
  - Интерактивное тестирование (Try it out)
  - Автоматическая генерация примеров
  - Скачивание OpenAPI спецификации
  - Подсветка синтаксиса
  - Поиск по эндпоинтам

### OpenAPI Спецификация

Полная спецификация API в формате OpenAPI 3.0.3.

- **URL**: http://localhost:8080/api/openapi.yaml
- **Формат**: YAML
- **Версия**: 2.0.0

### Генерация клиентских библиотек

Используйте OpenAPI спецификацию для генерации клиентских библиотек.

#### TypeScript Client

```bash
# Установка OpenAPI Generator
npm install -g @openapitools/openapi-generator-cli

# Генерация TypeScript клиента
openapi-generator-cli generate \
  -i http://localhost:8080/api/openapi.yaml \
  -g typescript-axios \
  -o ./generated-client
```

#### Java Client

```bash
openapi-generator-cli generate \
  -i http://localhost:8080/api/openapi.yaml \
  -g java \
  -o ./generated-client
```

#### Python Client

```bash
openapi-generator-cli generate \
  -i http://localhost:8080/api/openapi.yaml \
  -g python \
  -o ./generated-client
```

### Postman Collection

Импортируйте OpenAPI спецификацию в Postman:

1. Откройте Postman
2. File → Import
3. Выберите URL спецификации: `http://localhost:8080/api/openapi.yaml`
4. Postman автоматически создаст коллекцию

### cURL Examples

Все примеры в документации представлены в формате cURL для быстрого тестирования.

---

## Лимиты и ограничения

### Rate Limiting

В текущей версии нет ограничений на количество запросов.

### Размеры файлов

| Тип файла | Максимальный размер | Форматы |
|------------|-------------------|----------|
| Аудиофайл | 100 MB | MP3, WAV, FLAC |
| Изображение | 10 MB | JPG, PNG, GIF, WebP |

### Пагинация

- Минимум элементов на странице: 1
- Максимум элементов на странице: 100
- По умолчанию: 20 элементов

### JWT Токен

- Время жизни: 24 часа
- Алгоритм подписи: HS256
- Обновление: требуется повторная авторизация

---

## OAuth Конфигурация

### Google OAuth

1. **Authorization Endpoint**: `https://accounts.google.com/o/oauth2/v2/auth`
2. **Token Endpoint**: `https://oauth2.googleapis.com/token`
3. **User Info Endpoint**: `https://www.googleapis.com/oauth2/v2/userinfo`

### Yandex OAuth

1. **Authorization Endpoint**: `https://oauth.yandex.ru/authorize`
2. **Token Endpoint**: `https://oauth.yandex.ru/token`
3. **User Info Endpoint**: `https://login.yandex.ru/info?format=json`

---

## Мониторинг

### Логи

Приложение логирует все запросы и ошибки.

### Health Check

Используйте `/health` для проверки работоспособности:

```bash
curl http://localhost:8080/health
```

Ожидаемый ответ: `OK`

---

## Дополнительная документация

- [README.md](README.md) - Общая документация проекта
- [OAUTH_SETUP.md](OAUTH_SETUP.md) - Настройка OAuth
- [ADMIN_GUIDE.md](ADMIN_GUIDE.md) - Руководство администратора
- [MIGRATIONS.md](MIGRATIONS.md) - Система миграций
- [QUICK_START.md](QUICK_START.md) - Быстрый старт

---

## Поддержка

По вопросам и предложениям обращайтесь:
- GitHub Issues: https://gitflic.ru/project/koteyye/koteyye_music_be/issues
- Email: support@koteyye-music.com

---

**Версия API**: 2.0.0  
**Последнее обновление**: 2024-01-15  
**Автор**: Koteyye Music Team