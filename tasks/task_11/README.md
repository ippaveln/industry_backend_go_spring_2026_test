# Мини “production-like” сервис: in-memory repository + HTTP API + конкурентная безопасность
Сущность:

```go
type Task struct{ ID string; Title string; Done bool; UpdatedAt time.Time }
```

Интерфейс хранилища:

```go
type TaskRepo interface{ Create(title string) (Task, error); Get(id string) (Task, bool); List() []Task; SetDone(id string, done bool) (Task, error) }
```

HTTP:
- POST /tasks → создаёт
- GET /tasks/{id}
- GET /tasks
- PATCH /tasks/{id} (done true/false)


Требования: repo потокобезопасный (sync.RWMutex), UpdatedAt через внедряемые часы, JSON-валидация входа, корректные коды (400/404/200/201).
Юнит-тесты: repo гонки (хотя бы параллельные тесты), HTTP через httptest, стабильность времени через фейковые часы, сценарии ошибок.

![task 11](../../badges/tasks/task_11.svg)

// TODO!!!