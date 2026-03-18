# Мультиязычность (i18n) — Руководство для фронтенда

## Как работает

Бэкенд поддерживает **2 языка** для сообщений об ошибках:
- **Русский (`ru`)** — по умолчанию
- **Туркменский (`tk`)**

Язык выбирается через HTTP заголовок `Accept-Language`.

---

## Как использовать

### Отправка заголовка

Добавьте `Accept-Language` в каждый запрос:

```js
// Русский (или не отправлять заголовок — русский по умолчанию)
fetch('/api/sales', {
  headers: {
    'Authorization': 'Bearer <token>',
    'Accept-Language': 'ru'
  }
})

// Туркменский
fetch('/api/sales', {
  headers: {
    'Authorization': 'Bearer <token>',
    'Accept-Language': 'tk'
  }
})
```

### Axios — глобальная настройка

```js
import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:8080',
})

// Установить язык глобально
api.defaults.headers.common['Accept-Language'] = 'tk' // или 'ru'

// Или динамически через interceptor
api.interceptors.request.use((config) => {
  const lang = localStorage.getItem('language') || 'ru'
  config.headers['Accept-Language'] = lang
  return config
})
```

---

## Что переводится

### Переводится (поле `message` и `details`):
```json
{
  "success": false,
  "status_code": 400,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "код удаления обязателен",
    "details": {
      "warehouse_id": "поле обязательно"
    }
  }
}
```

### НЕ переводится (поле `code`):
Поле `code` всегда на английском — используйте его для логики в коде:

```js
if (error.response.data.error.code === 'VALIDATION_ERROR') {
  // показать ошибки полей
}
if (error.response.data.error.code === 'SALE_NOT_FOUND') {
  // перенаправить
}
```

---

## Примеры ответов

### Русский (`Accept-Language: ru`)
```json
{
  "success": false,
  "status_code": 400,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "неверный ID продажи"
  }
}
```

### Туркменский (`Accept-Language: tk`)
```json
{
  "success": false,
  "status_code": 400,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "nädogry satuw ID"
  }
}
```

### Без заголовка (русский по умолчанию)
```json
{
  "success": false,
  "status_code": 403,
  "error": {
    "code": "FORBIDDEN",
    "message": "неверный код удаления"
  }
}
```

---

## Валидация полей

При ошибках валидации `details` тоже переводится:

### Русский
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "ошибка валидации запроса",
    "details": {
      "warehouse_id": "поле обязательно",
      "username": "минимум 3 символов",
      "email": "должен быть действительный email"
    }
  }
}
```

### Туркменский
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "talabyň barlagy şowsuz",
    "details": {
      "warehouse_id": "meýdan zerur",
      "username": "iň az 3 simwol bolmaly",
      "email": "dogry email bolmaly"
    }
  }
}
```

---

## Переключение языка в UI

Рекомендуемый подход:

```js
// Хранить выбранный язык
function setLanguage(lang) {
  localStorage.setItem('language', lang) // 'ru' или 'tk'
}

function getLanguage() {
  return localStorage.getItem('language') || 'ru'
}
```

Пользователь выбирает язык в настройках → сохраняется в `localStorage` → отправляется с каждым запросом.

---

## Коды ошибок для логики (всегда английские)

| Код | HTTP | Описание |
|-----|------|----------|
| `VALIDATION_ERROR` | 400 | Невалидные данные |
| `UNAUTHORIZED` | 401 | Нет/невалидный токен |
| `FORBIDDEN` | 403 | Нет прав |
| `SALE_NOT_FOUND` | 404 | Продажа не найдена |
| `PRODUCT_NOT_FOUND` | 404 | Товар не найден |
| `NOT_FOUND` | 404 | Ресурс не найден |
| `CONFLICT` | 409 | Конфликт (дубликат) |
| `INTERNAL_ERROR` | 500 | Ошибка сервера |

Используйте `code` для if/switch логики, а `message` для показа пользователю.
