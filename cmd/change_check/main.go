package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"industry_backend_go/internal/config"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Change struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Raw    string `json:"raw"`
}

type Report struct {
	OK             bool     `json:"ok"`
	CheckedAt      string   `json:"checked_at"`
	DiffFile       string   `json:"diff_file"`
	ConfigFile     string   `json:"config_file"`
	AllowList      []string `json:"allow_list"`
	ChangedPaths   []string `json:"changed_paths"`
	Unexpected     []string `json:"unexpected"`
	UnexpectedBySt []Change `json:"unexpected_by_status,omitempty"`
}

func main() {
	cfgPath := flag.String("config", "./.etc/config.json", "config file")
	diffPath := flag.String("diff", "changed_files.raw", "path to diff file (prefer changed_files.raw)")
	outPath := flag.String("out", "change-policy-result.json", "output json file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(2)
	}
	matchers, err := compileAllowList(cfg.Diff.AllowList)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config read error:", err)
		os.Exit(2)
	}

	changes, err := readChanges(*diffPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "diff read error:", err)
		os.Exit(2)
	}

	changedSet := map[string]struct{}{}
	var unexpected []string
	unexpectedSet := map[string]struct{}{}
	var unexpectedBySt []Change

	addChanged := func(p string) {
		p = normalizePath(p)
		if p == "" {
			return
		}
		changedSet[p] = struct{}{}
		if !isAllowed(p, matchers) {
			if _, ok := unexpectedSet[p]; !ok {
				unexpectedSet[p] = struct{}{}
				unexpected = append(unexpected, p)
			}
		}
	}

	for _, ch := range changes {
		if ch.Path != "" {
			addChanged(ch.Path)
		}
		if ch.From != "" {
			addChanged(ch.From)
		}
		if ch.To != "" {
			addChanged(ch.To)
		}
	}

	changedPaths := make([]string, 0, len(changedSet))
	for p := range changedSet {
		changedPaths = append(changedPaths, p)
	}
	sort.Strings(changedPaths)
	sort.Strings(unexpected)

	// детализируем unexpectedBySt (чтобы было понятно, что именно случилось)
	for _, ch := range changes {
		paths := []string{}
		if ch.Path != "" {
			paths = append(paths, normalizePath(ch.Path))
		}
		if ch.From != "" {
			paths = append(paths, normalizePath(ch.From))
		}
		if ch.To != "" {
			paths = append(paths, normalizePath(ch.To))
		}

		bad := false
		for _, p := range paths {
			if p == "" {
				continue
			}
			if !isAllowed(p, matchers) {
				bad = true
				break
			}
		}
		if bad {
			unexpectedBySt = append(unexpectedBySt, ch)
		}
	}

	ok := len(unexpected) == 0

	rep := Report{
		OK:             ok,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		DiffFile:       *diffPath,
		ConfigFile:     *cfgPath,
		AllowList:      cfg.Diff.AllowList,
		ChangedPaths:   changedPaths,
		Unexpected:     unexpected,
		UnexpectedBySt: unexpectedBySt,
	}

	if *outPath != "" {
		if err := writeJSON(*outPath, rep); err != nil {
			fmt.Fprintln(os.Stderr, "report write error:", err)
			os.Exit(2)
		}
	}

	if ok {
		fmt.Printf("OK: all changes are allowed. Changed files: %d\n", len(changedPaths))
		os.Exit(0)
	}

	fmt.Printf("FAIL: unexpected changes detected: %d\n", len(unexpected))
	for _, p := range unexpected {
		fmt.Println(p)
	}
	os.Exit(1)
}

func writeJSON(p string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pathDir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func pathDir(p string) string {
	// маленький helper, чтобы не тянуть filepath (нам нужны / пути в репо)
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "."
	}
	if i == 0 {
		return "/"
	}
	return p[:i]
}

type matcher struct {
	pattern string
	re      *regexp.Regexp
}

func compileAllowList(patterns []string) ([]matcher, error) {
	out := make([]matcher, 0, len(patterns))
	for _, pat := range patterns {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		re, err := globToRegex(pat)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pat, err)
		}
		out = append(out, matcher{pattern: pat, re: re})
	}
	return out, nil
}

func isAllowed(p string, matchers []matcher) bool {
	if p == "" {
		return true
	}
	for _, m := range matchers {
		if m.re.MatchString(p) {
			return true
		}
	}
	return false
}

func globToRegex(pat string) (*regexp.Regexp, error) {
	pat = normalizePath(pat)
	if strings.HasSuffix(pat, "/") {
		// "dir/" => всё внутри
		pat = pat + "**"
	}

	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pat); i++ {
		ch := pat[i]

		if ch == '*' {
			// ** => match across slashes
			if i+1 < len(pat) && pat[i+1] == '*' {
				b.WriteString(".*")
				i++
				continue
			}
			// * => match within a path segment
			b.WriteString(`[^/]*`)
			continue
		}

		if ch == '?' {
			b.WriteString(`[^/]`)
			continue
		}

		// escape regexp metachars
		if strings.ContainsRune(`.+()|[]{}^$\/`, rune(ch)) {
			b.WriteByte('\\')
		}
		b.WriteByte(ch)
	}
	b.WriteString("$")

	return regexp.Compile(b.String())
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", "/")

	// часто из git diff вылезают префиксы a/ b/
	for {
		if strings.HasPrefix(p, "a/") || strings.HasPrefix(p, "b/") {
			p = p[2:]
			continue
		}
		break
	}

	// если diff делали между baseline/current, может прилипнуть префикс
	for _, pref := range []string{"./", "baseline/", "current/", "../baseline/"} {
		if strings.HasPrefix(p, pref) {
			p = strings.TrimPrefix(p, pref)
		}
	}

	p = strings.TrimPrefix(p, "/")
	p = path.Clean(p)
	if p == "." {
		return ""
	}
	return p
}

func readChanges(diffFile string) ([]Change, error) {
	f, err := os.Open(diffFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Change
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		ch, ok := parseDiffLine(raw)
		if !ok {
			// если формат непонятен — считаем как "изменённый файл = вся строка"
			p := normalizePath(line)
			if p != "" {
				out = append(out, Change{Status: "?", Path: p, Raw: raw})
			}
			continue
		}
		out = append(out, ch)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseDiffLine(raw string) (Change, bool) {
	line := strings.TrimRight(raw, "\r\n")
	var parts []string

	// предпочтительно таб-разделение (как в changed_files.raw)
	if strings.Contains(line, "\t") {
		parts = strings.Split(line, "\t")
	} else {
		parts = strings.Fields(line)
	}

	if len(parts) < 2 {
		return Change{}, false
	}

	st := parts[0]
	ch := Change{Status: st, Raw: raw}

	lead := st
	if len(lead) > 0 {
		lead = lead[:1]
	}

	if lead == "R" || lead == "C" {
		if len(parts) < 3 {
			return Change{}, false
		}
		ch.From = parts[1]
		ch.To = parts[2]
		return ch, true
	}

	ch.Path = parts[1]
	return ch, true
}
