package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Result struct {
	Status string `json:"status"`
}

type Shield struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

var taskRe = regexp.MustCompile(`task_(\d+)`)

func main() {
	inPath := flag.String("in", "package-results.json", "input json path")
	outDir := flag.String("out", "badges/tasks", "output dir")
	flag.Parse()

	b, err := os.ReadFile(*inPath)
	must(err)

	// Вход: map[".../task_00"] = {status: "..."}
	var m map[string]Result
	must(json.Unmarshal(b, &m))

	must(os.MkdirAll(*outDir, 0o755))

	// Чтобы файлы генерились стабильно (одинаковый порядок)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		r := m[k]

		taskID := taskIDFromKey(k) // "00", "01", ...
		label := "task " + taskID

		message, color := mapStatus(r.Status)

		out := Shield{
			SchemaVersion: 1,
			Label:         label,
			Message:       message,
			Color:         color,
		}

		filename := fmt.Sprintf("task_%s.json", taskID)
		outPath := filepath.Join(*outDir, filename)

		j, err := json.Marshal(out)
		must(err)
		j = append(j, '\n')

		must(os.WriteFile(outPath, j, 0o644))
	}

	fmt.Printf("generated %d badge json files in %s\n", len(keys), *outDir)
}

func taskIDFromKey(key string) string {
	// ключ типа "industry_backend_go/tasks/task_00"
	parts := strings.Split(key, "/")
	last := parts[len(parts)-1] // "task_00"
	m := taskRe.FindStringSubmatch(last)
	if len(m) == 2 {
		return m[1]
	}
	// запасной вариант, если формат внезапно другой
	fmt.Println("warning: cannot extract task ID from key:", key)
	return "??"
}

func mapStatus(s string) (message, color string) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pass":
		return "ok", "brightgreen"
	case "fail":
		return "fail", "red"
	default:
		// ты писал "unknow" — если прям так нужно, замени "unknown" на "unknow"
		return "unknown", "lightgrey"
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
