package config

type Config struct {
	Version string `json:"version"`
	Stream  string `json:"stream"`

	Tests struct {
		IgnorePackages []string `json:"ignore_packages"`
	} `json:"tests"`

	Diff struct {
		Original struct {
			Repo   string `json:"repo"`
			Branch string `json:"branch"`
		} `json:"original"`
		AllowList []string `json:"allow_list"`
	} `json:"diff"`
}
