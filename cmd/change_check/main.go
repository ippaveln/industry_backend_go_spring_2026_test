package main

import (
	"flag"
	"fmt"
	"industry_backend_go/internal/config"
	"os"
)

type Result struct {
	OK            bool     `json:"ok"`
	BaseRef       string   `json:"base_ref"`
	AllowlistFile []string `json:"allowlist"`
	Changed       []string `json:"changed"`
	Violations    []string `json:"violations"`
	Reason        string   `json:"reason,omitempty"`
}

func main() {
	base := flag.String("base", "master", "base branch name (master/main)")
	configFile := flag.String("config", "./.etc/config.json", "config file")
	out := flag.String("out", "change-policy-result.json", "output json file")
	fetch := flag.Bool("fetch", true, "git fetch origin <base> before diff")
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	allow := cfg.Diff.AllowList

	res := Result{
		BaseRef:       *base,
		AllowlistFile: allow,
	}

	_, _, _ = res, out, fetch
}
