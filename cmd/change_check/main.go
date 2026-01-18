package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Result struct {
	OK            bool     `json:"ok"`
	BaseRef       string   `json:"base_ref"`
	AllowlistFile string   `json:"allowlist_file"`
	Changed       []string `json:"changed"`
	Violations    []string `json:"violations"`
	Reason        string   `json:"reason,omitempty"`
}

func main() {
	base := flag.String("base", "master", "base branch name (master/main)")
	allow := flag.String("allow", "./.github/allowed_changes.txt", "allowlist file (paths/globs)")
	out := flag.String("out", "change-policy-result.json", "output json file")
	fetch := flag.Bool("fetch", true, "git fetch origin <base> before diff")
	flag.Parse()

	res := Result{
		BaseRef:       *base,
		AllowlistFile: *allow,
	}

	// 1) load allowlist patterns
	patterns, err := readAllowlist(*allow)
	if err != nil {
		res.OK = false
		res.Reason = err.Error()
		_ = writeResult(*out, res)
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}

	// 2) optional fetch base
	if *fetch {
		if err := run("git", "fetch", "origin", *base); err != nil {
			res.OK = false
			res.Reason = "git fetch failed: " + err.Error()
			_ = writeResult(*out, res)
			fmt.Fprintln(os.Stderr, "ERROR:", res.Reason)
			os.Exit(1)
		}
	}

	// 3) diff list
	changed, err := gitChangedFiles(*base)
	if err != nil {
		res.OK = false
		res.Reason = err.Error()
		_ = writeResult(*out, res)
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	res.Changed = changed

	// 4) enforce
	var violations []string
	for _, f := range changed {
		if !isAllowed(f, patterns) {
			violations = append(violations, f)
		}
	}
	res.Violations = violations
	res.OK = len(violations) == 0

	// 5) write result + also write changed list for удобства
	_ = os.WriteFile("changed_files.txt", []byte(strings.Join(changed, "\n")+"\n"), 0o644)

	if err := writeResult(*out, res); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: failed to write result:", err)
		os.Exit(1)
	}

	if res.OK {
		fmt.Println("OK: all changed files are allowed")
		os.Exit(0)
	} else {
		fmt.Println("FAIL: found disallowed changed files:")
		for _, v := range violations {
			fmt.Println(" -", v)
		}
		os.Exit(1)
	}
}

func readAllowlist(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("allowlist file not found: %s", path)
	}
	defer f.Close()

	var pats []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pats = append(pats, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("failed to read allowlist: %w", err)
	}
	return pats, nil
}

func gitChangedFiles(base string) ([]string, error) {
	// сравниваем base...HEAD (три точки) — это “изменения относительно общего предка”
	ref := "origin/" + base + "...HEAD"
	out, err := runOut("git", "diff", "--name-only", ref)
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var files []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// Нормализуем под '/'
		files = append(files, filepath.ToSlash(ln))
	}
	return files, nil
}

func isAllowed(file string, patterns []string) bool {
	f := filepath.ToSlash(file)

	for _, raw := range patterns {
		p := filepath.ToSlash(strings.TrimSpace(raw))

		// 1) exact match
		if p == f {
			return true
		}

		// 2) "dir/" treated as prefix
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(f, p) {
				return true
			}
			continue
		}

		// 3) "dir/**" treated as prefix
		if strings.HasSuffix(p, "/**") {
			prefix := strings.TrimSuffix(p, "**")
			if strings.HasPrefix(f, prefix) {
				return true
			}
			continue
		}

		// 4) simple glob (*, ?, [..]) via filepath.Match (works fine for usual patterns)
		if hasGlob(p) {
			ok, _ := filepath.Match(p, f)
			if ok {
				return true
			}
		}
	}
	return false
}

func hasGlob(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

func writeResult(path string, res Result) error {
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runOut(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}
