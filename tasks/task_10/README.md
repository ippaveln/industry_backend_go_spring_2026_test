# Мини “production-like” сервис: in-memory repository + HTTP API + конкурентная безопасность

![task 10](../../badges/tasks/task_10.svg)

Сделайте небольшой сервис задач: in-memory репозиторий + HTTP API. Решение будет проверяться тестами, поэтому важно строго следовать контракту ниже.

## Доменная модель


type Task struct {
 ID        string
 Title     string
 Done      bool
 UpdatedAt time.Time
}


## Контракт хранилища


type TaskRepo interface {
 Create(title string) (Task, error)
 Get(id string) (Task, bool)
 List() []Task
 SetDone(id string, done bool) (Task, error)
}


## HTTP API

### Эндпоинты

- POST /tasks → создать задачу  
- GET /tasks/{id} → получить задачу по id  
- GET /tasks → получить список задач  
- PATCH /tasks/{id} → обновить done (true/false)

### Форматы запросов

POST /tasks:


{"title":"buy milk"}


PATCH /tasks/{id}:


{"done":true}


### Форматы ответов

- Одна задача возвращается JSON-объектом с полями: id, title, done, updatedAt
- GET /tasks возвращает JSON-массив задач

## Требования к поведению

### 1) Потокобезопасность репозитория

- Реализация in-memory (например, map[string]Task)
- Защита через sync.RWMutex
- List() должен возвращать копию данных (чтобы внешний код не мог менять внутреннее состояние)

### 2) Время — только через внедрение часов

- Нельзя вызывать time.Now() напрямую внутри репозитория
- Сделайте интерфейс часов и внедрите его в репозиторий, например:


type Clock interface { Now() time.Time }


- UpdatedAt выставляется при Create и обновляется при SetDone

### 3) Валидация JSON и входных данных

- Невалидный JSON → 400 Bad Request
- Отсутствующие/неподходящие поля → 400 Bad Request
- Рекомендуется использовать json.Decoder + DisallowUnknownFields()
- title должен быть непустым после strings.TrimSpace

### 4) Корректные HTTP статусы

- POST /tasks → 201 Created
- Успешные GET / PATCH → 200 OK
- Не найдено (несуществующий id) → 404 Not Found
- Ошибка ввода/валидации → 400 Bad Request

### 5) Детерминированность списка

Чтобы ответы были стабильными, GET /tasks должен возвращать задачи в предсказуемом порядке:

- сортировка по UpdatedAt по убыванию (новые сначала)
- при равенстве UpdatedAt — по ID по возрастанию

### 6) ID

- ID должен быть уникальным в рамках процесса (можно счётчик/UUID — на ваше усмотрение)
- Пустой ID недопустим