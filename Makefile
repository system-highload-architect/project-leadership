# Makefile для управления проектом
.PHONY: help setup generate-diagrams docs validate clean

help:
	@echo "Доступные команды:"
	@echo "  make setup              - Установка всех зависимостей"
	@echo "  make generate-diagrams  - Генерация SVG из .puml файлов"
	@echo "  make docs               - Генерация HTML документации из Markdown"
	@echo "  make validate           - Проверка целостности документации"
	@echo "  make clean              - Очистка сгенерированных файлов"

setup:
	@echo "Установка зависимостей..."
	@./scripts/setup.sh

generate-diagrams:
	@echo "Генерация диаграмм из .puml файлов..."
	@./scripts/generate-diagrams.sh

docs:
	@echo "Генерация документации..."
	@mkdocs build --clean
	@echo "Документация собрана в директории site/"

validate:
	@echo "Проверка документации..."
	@./scripts/validate-docs.sh

clean:
	@echo "Очистка сгенерированных файлов..."
	@find . -name "*.svg" -type f -delete
	@rm -rf site/
	@echo "Готово."