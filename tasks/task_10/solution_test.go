package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{t: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Set(t time.Time) {
	c.mu.Lock()
	c.t = t
	c.mu.Unlock()
}

func (c *fakeClock) Add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func TestRepo_CreateGet(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)

	created, err := repo.Create("hello")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected non-empty ID")
	}
	if created.Title != "hello" {
		t.Fatalf("expected Title=hello, got %q", created.Title)
	}
	if created.Done {
		t.Fatalf("expected Done=false by default")
	}
	if !created.UpdatedAt.Equal(fc.Now()) {
		t.Fatalf("expected UpdatedAt=%v, got %v", fc.Now(), created.UpdatedAt)
	}

	got, ok := repo.Get(created.ID)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got.ID != created.ID || got.Title != created.Title || got.Done != created.Done || !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("unexpected task from Get: %+v", got)
	}
}

func TestRepo_Get_NotFound(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)

	_, ok := repo.Get("missing")
	if ok {
		t.Fatalf("expected ok=false")
	}
}

func TestRepo_SetDone_UpdatesTime(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)

	task, err := repo.Create("x")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	fc.Add(5 * time.Second)
	updated, err := repo.SetDone(task.ID, true)
	if err != nil {
		t.Fatalf("SetDone error: %v", err)
	}
	if !updated.Done {
		t.Fatalf("expected Done=true")
	}
	if !updated.UpdatedAt.Equal(fc.Now()) {
		t.Fatalf("expected UpdatedAt=%v, got %v", fc.Now(), updated.UpdatedAt)
	}
}

func TestRepo_SetDone_NotFound(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)

	_, err := repo.SetDone("missing", true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRepo_List_ReturnsCopy(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)

	a, _ := repo.Create("a")
	_, _ = repo.Create("b")

	list := repo.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}

	// Попробуем “испортить” срез/элемент снаружи.
	list[0].Title = "hacked"

	got, ok := repo.Get(a.ID)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got.Title != "a" {
		t.Fatalf("expected internal state not affected; got Title=%q", got.Title)
	}
}

func TestRepo_ConcurrentAccess_NoPanicsAndConsistentLen(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)

	const n = 200
	var wg sync.WaitGroup
	wg.Add(n)

	ids := make([]string, n)
	var idsMu sync.Mutex

	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()

			task, err := repo.Create("t")
			if err != nil {
				t.Errorf("Create error: %v", err)
				return
			}

			idsMu.Lock()
			ids[i] = task.ID
			idsMu.Unlock()

			// Параллельно дергаем разные методы.
			_, _ = repo.Get(task.ID)
			_ = repo.List()
			_, _ = repo.SetDone(task.ID, true)
		}()
	}

	wg.Wait()

	list := repo.List()
	if len(list) != n {
		t.Fatalf("expected %d tasks, got %d", n, len(list))
	}

}

type taskDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func do(t *testing.T, h http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func decodeJSON[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	dec := json.NewDecoder(r)
	var v T
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return v
}

func TestHTTP_Create_201_AndBody(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"buy milk"}`))
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	task := decodeJSON[taskDTO](t, rr.Body)
	if task.ID == "" {
		t.Fatalf("expected non-empty id")
	}
	if task.Title != "buy milk" {
		t.Fatalf("expected title=buy milk, got %q", task.Title)
	}
	if task.Done {
		t.Fatalf("expected done=false")
	}
	if !task.UpdatedAt.Equal(fc.Now()) {
		t.Fatalf("expected updatedAt=%v, got %v", fc.Now(), task.UpdatedAt)
	}
}

func TestHTTP_Create_Validation_BadJSON_400(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTP_Create_Validation_EmptyTitle_400(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"   "}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTP_Create_Validation_UnknownField_400(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	rr := do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"x","extra":1}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTP_GetByID_200_And404(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	created := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"x"}`)).Body)

	rr := do(t, h, http.MethodGet, "/tasks/"+created.ID, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	got := decodeJSON[taskDTO](t, rr.Body)
	if got.ID != created.ID {
		t.Fatalf("expected id=%s, got %s", created.ID, got.ID)
	}

	rr2 := do(t, h, http.MethodGet, "/tasks/missing", nil)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestHTTP_PatchDone_200_UpdatesTime(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	created := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"x"}`)).Body)

	fc.Add(10 * time.Second)
	rr := do(t, h, http.MethodPatch, "/tasks/"+created.ID, []byte(`{"done":true}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	updated := decodeJSON[taskDTO](t, rr.Body)
	if !updated.Done {
		t.Fatalf("expected done=true")
	}
	if !updated.UpdatedAt.Equal(fc.Now()) {
		t.Fatalf("expected updatedAt=%v, got %v", fc.Now(), updated.UpdatedAt)
	}
}

func TestHTTP_PatchDone_Validation_400(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	created := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"x"}`)).Body)

	cases := []string{
		`{"done":"true"}`,     // wrong type
		`{}`,                  // missing field
		`{"done":true,"x":1}`, // unknown field
		`{"done":`,            // bad json
	}
	for _, body := range cases {
		rr := do(t, h, http.MethodPatch, "/tasks/"+created.ID, []byte(body))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("body=%s: expected 400, got %d: %s", body, rr.Code, rr.Body.String())
		}
	}
}

func TestHTTP_PatchDone_404(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	rr := do(t, h, http.MethodPatch, "/tasks/missing", []byte(`{"done":true}`))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTP_List_200_SortedByUpdatedAtDesc_AndTieByIDAsc(t *testing.T) {
	t.Parallel()

	fc := newFakeClock(time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC))
	repo := NewInMemoryTaskRepo(fc)
	h := NewHTTPHandler(repo)

	// Разные UpdatedAt: создадим 3 задачи в разное время.
	t1 := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"a"}`)).Body)
	fc.Add(1 * time.Second)
	t2 := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"b"}`)).Body)
	fc.Add(1 * time.Second)
	t3 := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"c"}`)).Body)

	rr := do(t, h, http.MethodGet, "/tasks", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	list := decodeJSON[[]taskDTO](t, rr.Body)
	if len(list) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(list))
	}
	// Должно быть: t3, t2, t1 по времени.
	if list[0].ID != t3.ID || list[1].ID != t2.ID || list[2].ID != t1.ID {
		t.Fatalf("expected order t3,t2,t1; got ids: %v,%v,%v", list[0].ID, list[1].ID, list[2].ID)
	}

	// Tie-break: одинаковое UpdatedAt у нескольких задач => сортировка по ID ASC.
	fc.Set(time.Date(2026, 1, 24, 13, 0, 0, 0, time.UTC))
	a := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"tie1"}`)).Body)
	b := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"tie2"}`)).Body)
	c := decodeJSON[taskDTO](t, do(t, h, http.MethodPost, "/tasks", []byte(`{"title":"tie3"}`)).Body)

	rr2 := do(t, h, http.MethodGet, "/tasks", nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	list2 := decodeJSON[[]taskDTO](t, rr2.Body)

	// Возьмём только три “tie” задачи из списка и проверим порядок по ID.
	var gotIDs []string
	wantIDs := []string{a.ID, b.ID, c.ID}
	sort.Strings(wantIDs)

	for _, x := range list2 {
		if x.UpdatedAt.Equal(fc.Now()) && (x.ID == a.ID || x.ID == b.ID || x.ID == c.ID) {
			gotIDs = append(gotIDs, x.ID)
		}
	}
	if len(gotIDs) != 3 {
		t.Fatalf("expected to find 3 tie tasks in list, found %d", len(gotIDs))
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("expected tie order by id asc=%v, got=%v", wantIDs, gotIDs)
		}
	}
}
