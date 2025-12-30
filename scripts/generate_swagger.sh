#!/bin/bash

# Скрипт для генерации Swagger документации с примерами

set -e  # Остановить при ошибке

echo "🚀 Начинаю генерацию Swagger документации..."

# Переходим в корень проекта
cd "$(dirname "$0")/.."

# 1. Генерируем базовую документацию
echo "📝 Генерирую базовую документацию..."
swag init -g cmd/api/main.go -o docs --parseInternal

# 2. Добавляем примеры
echo "✨ Добавляю примеры ответов..."
go run scripts/add_swagger_examples.go docs/swagger.json

# 3. Обновляем docs.go и swagger.yaml
echo "🔄 Обновляю docs.go и swagger.yaml..."
swag init -g cmd/api/main.go -o docs --parseInternal

# 4. Снова добавляем примеры (чтобы docs.json тоже были обновлены)
echo "✨ Добавляю примеры в финальную версию..."
go run scripts/add_swagger_examples.go docs/swagger.json

# 5. Проверяем компиляцию
echo "🔍 Проверяю компиляцию..."
if go build -o /tmp/koteyye_music_be ./cmd/api; then
    echo "✅ Компиляция успешна!"
    rm -f /tmp/koteyye_music_be
else
    echo "❌ Ошибка компиляции!"
    exit 1
fi

echo ""
echo "🎉 Swagger документация успешно сгенерирована!"
echo ""
echo "📁 Созданные файлы:"
echo "  - docs/docs.go"
echo "  - docs/swagger.json"
echo "  - docs/swagger.yaml"
echo ""
echo "🌐 Доступ к Swagger UI:"
echo "  http://localhost:8080/swagger/index.html"
echo ""
echo "✅ Все примеры добавлены!"
