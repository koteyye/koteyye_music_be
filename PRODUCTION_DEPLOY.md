# Production Deployment Guide

## Быстрый запуск

1. **Скопируйте .env файл:**
```bash
cp .env.production .env
```

2. **Отредактируйте .env файл:**
- Установите надежные пароли для `DB_PASSWORD`, `MINIO_SECRET_KEY`
- Сгенерируйте случайный `JWT_SECRET` (минимум 32 символа)
- При необходимости добавьте OAuth credentials

3. **Запустите все сервисы:**
```bash
docker-compose -f docker-compose.prod.yml up -d
```

4. **Проверьте что все работает:**
```bash
# Проверить статус сервисов
docker-compose -f docker-compose.prod.yml ps

# Проверить логи
docker-compose -f docker-compose.prod.yml logs backend

# Тест API
curl http://localhost:8080/api/tracks
```

## Что включено

- **PostgreSQL 16** - основная база данных с автоматическими миграциями
- **MinIO** - S3-совместимое хранилище для аудио и изображений
- **Backend API** - Go сервис с полным функционалом

## Порты

- `8080` - Backend API
- `5432` - PostgreSQL (для внешних подключений)
- `9000` - MinIO API
- `9001` - MinIO Console

## Volumes

- `postgres_data` - данные PostgreSQL
- `minio_data` - файлы MinIO

## Healthchecks

Все сервисы имеют healthcheck'и для корректного порядка запуска:
1. Сначала стартуют PostgreSQL и MinIO
2. Создается bucket в MinIO
3. Запускается Backend после готовности зависимостей

## Остановка

```bash
docker-compose -f docker-compose.prod.yml down
```

## Полная очистка (ВНИМАНИЕ: удалит все данные)

```bash
docker-compose -f docker-compose.prod.yml down -v
```

## Обновление

```bash
# Остановить сервисы
docker-compose -f docker-compose.prod.yml down

# Обновить код
git pull

# Пересобрать и запустить
docker-compose -f docker-compose.prod.yml up -d --build
```