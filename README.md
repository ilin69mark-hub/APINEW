# Микросервисная архитектура на Go

## Описание

Проект представляет собой микросервисную архитектуру, состоящую из трех основных сервисов:

1. **API Gateway** (порт 8080) - единая точка входа для клиентов
2. **Comment Service** (порт 8081) - управление комментариями
3. **Censor Service** (порт 8082) - проверка комментариев на запрещенные слова
4. **News Aggregator** (порт 8083) - заглушка сервиса новостей

## Архитектура

```
Клиент → APIGateway → [CensorService] → CommentService
                    ↘ [NewsAggregator*] ↗
* NewsAggregator предполагается внешним сервисом
```

## Установка и запуск

### Локальный запуск

1. Убедитесь, что у вас установлен Go 1.21+
2. Установите зависимости:

```bash
make deps
```

3. Соберите и запустите все сервисы:

```bash
make build
make run
```

### Запуск через Docker

```bash
# Сборка и запуск всех сервисов
make docker-run

# Или напрямую через docker-compose
docker-compose up --build
```

## Тестирование

### Запуск тестов

```bash
make test
```

### Использование Postman коллекции

В корне проекта находится файл `news_microservices.postman_collection.json`, который можно импортировать в Postman для тестирования API.

## API Endpoints

### API Gateway (порт 8080)

- `GET /health` - проверка работоспособности
- `GET /news` - получить все новости с пагинацией
- `GET /news/{id}` - получить новость по ID с комментариями
- `POST /comment` - создать комментарий (проходит через цензуру)

### Comment Service (порт 8081)

- `GET /health` - проверка работоспособности
- `GET /comments?news_id={id}` - получить комментарии для новости
- `POST /comments` - создать комментарий
- `DELETE /comments/{id}` - удалить комментарий

### Censor Service (порт 8082)

- `GET /health` - проверка работоспособности
- `POST /check` - проверить текст на наличие запрещенных слов

### News Aggregator (порт 8083)

- `GET /health` - проверка работоспособности
- `GET /news` - получить все новости
- `GET /news/{id}` - получить новость по ID

## Особенности реализации

- Все сервисы имеют структурированное логирование с ID запроса
- Реализована проверка на запрещенные слова (qwerty, йцукен, zxvbnm)
- Поддержка пагинации и поиска в новостях
- Валидация входных данных
- Обработка ошибок с единым форматом ответа

## Структура проекта

```
/workspace/
├── api-gateway/          # API Gateway сервис
│   ├── main.go           # Основной файл
│   ├── go.mod            # Зависимости
│   └── Dockerfile        # Для контейнеризации
├── comment-service/      # Сервис комментариев
│   ├── main.go           # Основной файл
│   ├── go.mod            # Зависимости
│   └── Dockerfile        # Для контейнеризации
├── censor-service/       # Сервис цензуры
│   ├── main.go           # Основной файл
│   ├── go.mod            # Зависимости
│   └── Dockerfile        # Для контейнеризации
├── news-aggregator/      # Заглушка сервиса новостей
│   ├── main.go           # Основной файл
│   ├── go.mod            # Зависимости
│   └── Dockerfile        # Для контейнеризации
├── docker-compose.yml    # Конфигурация для запуска всех сервисов
├── Makefile              # Команды для сборки и запуска
└── news_microservices.postman_collection.json  # Postman коллекция для тестирования
```

## Make команды

- `make build` - собрать все сервисы
- `make run` - запустить все сервисы
- `make test` - запустить тесты для всех сервисов
- `make clean` - очистить артефакты сборки
- `make docker-build` - собрать Docker образы
- `make docker-run` - запустить все сервисы в Docker
- `make docker-down` - остановить все Docker контейнеры
- `make deps` - установить зависимости для всех сервисов
```

## Flow создания комментария

1. Клиент → POST /comment (APIGateway)
2. APIGateway → POST /check (CensorService) с текстом
3. Если 400 → ошибка клиенту
4. Если 200 → APIGateway → POST /comments (CommentService)
5. CommentService сохраняет в БД
6. Успешный ответ клиенту

## Flow получения новости

1. Клиент → GET /news/{id} (APIGateway)
2. APIGateway параллельно:
   - Запрос деталей новости (NewsAggregator)
   - GET /comments?news_id={id} (CommentService)
3. Агрегация результатов
4. Ответ клиенту