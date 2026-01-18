package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	Status string `json:"status"`
}

type Task struct {
	Key    string // исходный ключ из json
	Num    int    // для сортировки
	ID     string // "00"
	Status string
}

var taskRe = regexp.MustCompile(`task_(\d+)`)

func main() {
	inPath := flag.String("in", "package-results.json", "input json path")
	outDir := flag.String("out", "badges/tasks", "output directory for .svg files")
	style := flag.String("style", "flat", "shields style (flat, flat-square, for-the-badge, etc.)")
	unknownMsg := flag.String("unknown", "unknown", "message for unknown status (e.g. unknown or unknow)")
	timeout := flag.Duration("timeout", 20*time.Second, "http timeout")
	flag.Parse()

	b, err := os.ReadFile(*inPath)
	must(err)

	var m map[string]Result
	must(json.Unmarshal(b, &m))

	tasks := make([]Task, 0, len(m))
	for k, r := range m {
		id, num, ok := extractTaskID(k)
		if !ok {
			// если в json есть ключи не про task_XX — пропускаем
			continue
		}
		tasks = append(tasks, Task{
			Key:    k,
			Num:    num,
			ID:     id,
			Status: r.Status,
		})
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Num != tasks[j].Num {
			return tasks[i].Num < tasks[j].Num
		}
		return tasks[i].Key < tasks[j].Key
	})

	must(os.MkdirAll(*outDir, 0o755))

	client := &http.Client{Timeout: *timeout}

	written := 0
	for _, t := range tasks {
		msg, color := mapStatus(t.Status, *unknownMsg)
		label := "task " + t.ID

		badgeURL := buildBadgeURL(label, msg, color, *style)

		outPath := filepath.Join(*outDir, fmt.Sprintf("task_%s.svg", t.ID))
		must(downloadToFile(client, badgeURL, outPath))

		written++
	}

	fmt.Printf("generated %d svg badge files in %s\n", written, *outDir)
}

func extractTaskID(key string) (id string, num int, ok bool) {
	m := taskRe.FindStringSubmatch(key)
	if len(m) != 2 {
		return "", 0, false
	}
	id = m[1] // сохраняем как есть (с ведущими нулями)
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return "", 0, false
	}
	return id, n, true
}

func mapStatus(status, unknownMsg string) (message, color string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass":
		return "ok", "brightgreen"
	case "fail":
		return "fail", "red"
	default:
		return unknownMsg, "lightgrey"
	}
}

func buildBadgeURL(label, message, color, style string) string {
	// Важно: в /badge/ используется формат LABEL-MESSAGE-COLOR.
	// Экранируем каждый сегмент отдельно, чтобы пробелы стали %20.
	l := url.PathEscape(label)
	m := url.PathEscape(message)
	c := url.PathEscape(color)

	u := fmt.Sprintf("https://img.shields.io/badge/%s-%s-%s.svg", l, m, c)

	v := url.Values{}
	if style != "" {
		v.Set("style", style)
	}
	// Можно добавить cacheSeconds, если хочешь:
	// v.Set("cacheSeconds", "60")

	if qs := v.Encode(); qs != "" {
		u += "?" + qs
	}
	return u
}

func downloadToFile(client *http.Client, u, outPath string) error {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "badgesvg/1.0 (+github actions)")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// чтобы увидеть текст ошибки shields, но не читать бесконечно
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("GET %s: status %d: %s", u, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	tmp := outPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	return os.Rename(tmp, outPath)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
