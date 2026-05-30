package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Base16Theme struct {
	Name   string `json:"name"`
	Base00 string `json:"base00"`
	Base01 string `json:"base01"`
	Base02 string `json:"base02"`
	Base03 string `json:"base03"`
	Base04 string `json:"base04"`
	Base05 string `json:"base05"`
	Base06 string `json:"base06"`
	Base07 string `json:"base07"`
	Base08 string `json:"base08"`
	Base09 string `json:"base09"`
	Base0A string `json:"base0A"`
	Base0B string `json:"base0B"`
	Base0C string `json:"base0C"`
	Base0D string `json:"base0D"`
	Base0E string `json:"base0E"`
	Base0F string `json:"base0F"`
}

var DefaultDarkTheme = Base16Theme{
	Name:   "default-dark",
	Base00: "#181818",
	Base01: "#282828",
	Base02: "#383838",
	Base03: "#585858",
	Base04: "#b8b8b8",
	Base05: "#d8d8d8",
	Base06: "#e8e8e8",
	Base07: "#f8f8f8",
	Base08: "#ab4642",
	Base09: "#dc9656",
	Base0A: "#f7ca88",
	Base0B: "#a1b56c",
	Base0C: "#86c1b9",
	Base0D: "#7cafc2",
	Base0E: "#ba8baf",
	Base0F: "#a16946",
}

func LoadTheme(cfg Config) Base16Theme {
	themePath := filepath.Join(cfg.CacheDir, "theme.json")
	data, err := os.ReadFile(themePath)
	if err != nil {
		return DefaultDarkTheme
	}
	var theme Base16Theme
	if err := json.Unmarshal(data, &theme); err != nil {
		return DefaultDarkTheme
	}
	return theme
}

func (t Base16Theme) ToCSS() string {
	return fmt.Sprintf(`:root {
  --base00: %s;
  --base01: %s;
  --base02: %s;
  --base03: %s;
  --base04: %s;
  --base05: %s;
  --base06: %s;
  --base07: %s;
  --base08: %s;
  --base09: %s;
  --base0A: %s;
  --base0B: %s;
  --base0C: %s;
  --base0D: %s;
  --base0E: %s;
  --base0F: %s;
}`, t.Base00, t.Base01, t.Base02, t.Base03, t.Base04, t.Base05,
		t.Base06, t.Base07, t.Base08, t.Base09, t.Base0A, t.Base0B,
		t.Base0C, t.Base0D, t.Base0E, t.Base0F)
}
