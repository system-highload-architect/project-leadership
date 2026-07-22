#!/bin/bash
# Проверка целостности документации

set -e

echo "Проверка Markdown файлов..."
find docs -name "*.md" -exec markdown-link-check -q {} \;

echo "Проверка наличия обязательных файлов..."
MISSING=0

MANDATORY_FILES=(
    "docs/README.md"
    "docs/strategic/README.md"
    "docs/architecture/README.md"
    "docs/architecture/adr/README.md"
    "docs/architecture/api-contracts/openapi.yaml"
    "docs/architecture/data-model/schema.sql"
    "docs/project-management/README.md"
    "docs/people-processes/README.md"
    "docs/operations/README.md"
    "docs/templates/README.md"
)

for file in "${MANDATORY_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "  ❌ Отсутствует: $file"
        MISSING=1
    else
        echo "  ✅ $file"
    fi
done

if [ $MISSING -eq 1 ]; then
    echo "❌ Некоторые обязательные файлы отсутствуют."
    exit 1
else
    echo "✅ Все обязательные файлы на месте."
fi