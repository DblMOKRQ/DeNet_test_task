# DeNet_test_task

Это REST API сервис для управления баллами пользователей, заданиями и реферальными связями.

## Возможности

- Регистрация и аутентификация пользователей с JWT
    
- Управление профилями пользователей
    
- Выполнение заданий с начислением баллов
    
- Реферальная система с бонусными баллами
    
- Таблица лидеров по количеству баллов
    

## Требования

- Установленные Docker и Docker Compose
    
- Go 1.16+ (если запускать локально без Docker)
    

## Запуск проекта

### С использованием Docker (Рекомендуется)

1. Клонируйте репозиторий:
	```bash
	git clone https://github.com/your-repo/user-points-system.git
	cd user-points-system
```
2. Соберите и запустите сервисы:
	```bash
	docker-compose up -d --build
```
### Без Docker

1. Установите PostgreSQL и создайте базу данных `user_points`
    
2. Обновите файл `config.yaml` с вашими учетными данными БД
    
3. Запустите сервер:
	```bash
	go run main.go
```

## API Эндпоинты

### Публичные эндпоинты

- `POST /users/register` - Регистрация нового пользователя
```json
{
  "username": "testuser",
  "password": "password123"
}
```
### Защищенные эндпоинты (требуют JWT в заголовке Authorization)

- `GET /users/status` - Получить статус текущего пользователя
    
- `GET /users/leaderboard?limit=10` - Получить таблицу лидеров (по умолчанию 10)
    
- `POST /users/task/complete` - Выполнить задание
```json
{
  "task_type": "vk",
  "points": 50
}
```

- `POST /users/referrer` - Добавить реферера
```json
{
  "referrer_id": "uuid-реферера"
}
```
