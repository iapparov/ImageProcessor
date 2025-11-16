# ImageProcessor

## Описание

ImageProcessor — сервис, который принимает изображения, кладёт задачу на обработку в очередь (через Apache Kafka), и уже в фоне обрабатывает файл (например, делает resize или ставит watermark)

## Состав репозитория

- **cmd/imageProcessor/main.go** — точка входа через FX DI.
- **internal/**
  - **app/** — реализация бизнес-логики (ImageService).
  - **config/** — загрузка конфигурации из YAML.
  - **di/** — реализация зависимостей через UberFX.
  - **domain/** — Модель изображения (Image).
  - **imgprocessor/** — обработка изображения.
  - **broker/kafka_consumer** — работа с Kafka (consumer).
  - **broker/kafka_producer** — работа с Kafka (producer).
  - **storage/db/** — работа с PostgreSQL (CRUD).
  - **web/** — HTTP-обработчики и роутер.
- **config/local.yaml** — пример конфигурации.
- **migrations/** — SQL-миграции для PostgreSQL.
- **docs/** — Swagger-документация.
- **web/index.html** — простая страница для отпарвки и редактирования изображения через API.
- **docker-compose.yml** — запуск PostgreSQL, Kafka через Docker.
- **.env.example** - пример env файла для кредов.

---

## Быстрый старт

### 1. Запуск инфраструктуры

```sh
docker-compose up -d
```
(Запустит контейнеры: postgres → порт 5433, kafka → порт 9092)

### 2. Настроить переменные окружения и конфигурацию
(пример в .env.example + config/local.yaml)

### 3. Применить миграции (migrate):

```sh
migrate -path migrations -database "postgres://user:password@localhost:5433/dbname?sslmode=disable" up
```

### 4. Запуск сервиса

```sh
go run ./cmd/imageProcessor/main.go
```

Сервис стартует на порту 8080.


## API

- **POST /api/upload** — загрузка изображения на обработку (FORM: file, resize, mini, watermark);
- **GET /api/image/{id}** — получение обработанного изображения;
- **DELETE /api/image/{id}** —  удаление изображения;
- **Swagger**: [http://localhost:8080/api/swagger/index.html](http://localhost:8080/api/swagger/index.html)

---

## Веб-интерфейс
Откройте index.html в браузере — простая страница для отпарвки и редактирования изображения через API.


## Тесты
Юнит-тесты: `go test ./internal/...`

## Миграции

- `migrations/000001_create_tables.up.sql` — создание таблиц.
- `migrations/000001_create_tables.down.sql` — удаление таблиц.

---

## Логирование и метрики
Логирование реализовано через wbf/zlog (используется в internal/*).

## Зависимости

- Go 1.25+
- PostgreSQL 16+
- Kafka 7.5+
- Zookeeper 7.5+
- Docker (для локального запуска инфраструктуры)

---

## Swagger

- Swagger: [docs/swagger.yaml](docs/swagger.yaml)
- Документация генерируется автоматически и доступна по `/swagger/*`.