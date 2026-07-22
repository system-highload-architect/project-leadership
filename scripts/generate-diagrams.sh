#!/bin/bash
# Генерация SVG из всех .puml файлов в проекте

set -e

echo "Поиск .puml файлов..."
PUML_FILES=$(find . -name "*.puml" -not -path "./site/*" -not -path "./.venv/*")

if [ -z "$PUML_FILES" ]; then
    echo "Нет .puml файлов для обработки."
    exit 0
fi

echo "Найдено .puml файлов:"
echo "$PUML_FILES"

for file in $PUML_FILES; do
    echo "Обработка: $file"
    plantuml -tsvg "$file"
done

echo "Генерация завершена."