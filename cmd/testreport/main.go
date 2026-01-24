package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"industry_backend_go/internal/config"
	"os"
	"strings"
)

type TestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test,omitempty"`
	Output  string  `json:"Output,omitempty"`
	Elapsed float64 `json:"Elapsed,omitempty"`
}

type PackageResult struct {
	Status      string   `json:"status"` // pass|fail|skip|unknown
	FailedTests []string `json:"failed_tests,omitempty"`
}

func loadPackages(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	var pkgs []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			pkgs = append(pkgs, l)
		}
	}
	return pkgs, nil
}

func ignoredPackage(configPath *string) map[string]struct{} {
	if configPath == nil {
		fmt.Fprintf(os.Stderr, "non parse config path\n")
		os.Exit(2)
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create cfg: %v\n", err)
		os.Exit(2)
	}
	ignoredPkgsLines := cfg.Tests.IgnorePackages
	ignoredPkgs := make(map[string]struct{}, len(ignoredPkgsLines))
	for _, l := range ignoredPkgsLines {
		l = strings.TrimSpace(l)
		if l != "" {
			ignoredPkgs[l] = struct{}{}
		}
	}
	return ignoredPkgs
}

func main() {
	inPath := flag.String("in", "", "input file (go test -json output). If empty: read stdin")
	outPath := flag.String("out", "package-results.json", "output json file")
	pkgsPath := flag.String("pkgs", "", "optional packages list file (one package per line), e.g. from `go list ./...`")
	configPath := flag.String("config", "./.etc/config.json", "config file")
	flag.Parse()

	pkgs, err := loadPackages(*pkgsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load pkgs: %v\n", err)
		os.Exit(2)
	}

	ignoredPkgs := ignoredPackage(configPath)
	_ = ignoredPkgs

	results := map[string]*PackageResult{}
	ensure := func(pkg string) {
		if pkg == "" {
			return
		}
		if _, ok := ignoredPkgs[pkg]; ok {
			return
		}
		if _, ok := results[pkg]; !ok {
			results[pkg] = &PackageResult{Status: "unknown"}
		}
	}

	// prefill expected packages (so they appear even if no events were emitted)
	for _, p := range pkgs {
		ensure(p)
	}

	var in *os.File
	if *inPath == "" {
		in = os.Stdin
	} else {
		f, err := os.Open(*inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open input: %v\n", err)
			os.Exit(2)
		}
		defer f.Close()
		in = f
	}

	sc := bufio.NewScanner(in)
	// go test output lines can be large (panic stacktrace, long logs)
	sc.Buffer(make([]byte, 1024), 10*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var ev TestEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			// ignore non-json garbage lines
			continue
		}

		if ev.Package == "" {
			continue
		}
		ensure(ev.Package)

		// package-level result: Action pass/fail/skip and empty Test
		if ev.Test == "" {
			if _, ignored := ignoredPkgs[ev.Package]; ignored {
				continue
			}
			switch ev.Action {
			case "pass":
				results[ev.Package].Status = "pass"
			case "fail":
				results[ev.Package].Status = "fail"
			case "skip":
				// sometimes packages get skipped; keep it explicit
				if results[ev.Package].Status == "unknown" {
					results[ev.Package].Status = "skip"
				}
			}
			continue
		}

		// test-level fail: Action fail and Test present
		if ev.Action == "fail" && ev.Test != "" {
			results[ev.Package].FailedTests = append(results[ev.Package].FailedTests, ev.Test)
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scan input: %v\n", err)
		os.Exit(2)
	}

	out, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(2)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(2)
	}
}
