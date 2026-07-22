#!/bin/bash
# Установка всех зависимостей

set -e

echo "Установка зависимостей для проекта..."

# Проверка наличия plantuml
if ! command -v plantuml &> /dev/null; then
    echo "⚠️ PlantUML не найден. Устанавливаем через brew (macOS) или apt (Linux)..."
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        sudo apt-get update && sudo apt-get install -y plantuml graphviz
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        brew install plantuml graphviz
    else
        echo "❌ Не удалось определить ОС. Установите PlantUML и Graphviz вручную."
        exit 1
    fi
else
    echo "✅ PlantUML найден."
fi

# Проверка наличия mkdocs
if ! command -v mkdocs &> /dev/null; then
    echo "⚠️ mkdocs не найден. Устанавливаем через pip..."
    pip install mkdocs mkdocs-material
else
    echo "✅ mkdocs найден."
fi

# Проверка наличия markdown-link-check
if ! command -v markdown-link-check &> /dev/null; then
    echo "⚠️ markdown-link-check не найден. Устанавливаем через npm..."
    npm install -g markdown-link-check
else
    echo "✅ markdown-link-check найден."
fi

echo "✅ Все зависимости установлены."