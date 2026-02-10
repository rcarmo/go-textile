package gotextile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type fixtureEntry struct {
	Input    string                   `yaml:"input"`
	Expect   string                   `yaml:"expect"`
	Setup    []map[string]interface{} `yaml:"setup"`
	Assert   string                   `yaml:"assert"`
	Doctype  string                   `yaml:"doctype"`
	Class    string                   `yaml:"class"`
}

func TestFixtures(t *testing.T) {
	root := "test/fixtures"
	filter := strings.TrimSpace(os.Getenv("TEXTILE_FIXTURE_FILTER"))
	limit := toInt(os.Getenv("TEXTILE_FIXTURE_LIMIT"))
	count := 0
	files, err := filepath.Glob(filepath.Join(root, "*.yaml"))
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no fixture files found in %s", root)
	}
	sort.Strings(files)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		entries := map[string]fixtureEntry{}
		if err := yaml.Unmarshal(content, &entries); err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		entryNames := make([]string, 0, len(entries))
		for name := range entries {
			entryNames = append(entryNames, name)
		}
		sort.Strings(entryNames)
		for _, name := range entryNames {
			entry := entries[name]
			fixtureName := fmt.Sprintf("%s/%s", filepath.Base(file), name)
			if filter != "" && !strings.Contains(fixtureName, filter) {
				continue
			}
			if limit > 0 && count >= limit {
				return
			}
			fixtureOptions := optionsFromSetup(entry.Setup)
			if strings.Contains(entry.Class, "DisableSymbols") {
				fixtureOptions.NoGlyphs = true
			}
			if strings.EqualFold(entry.Doctype, "html5") {
				fixtureOptions.HTML5 = true
			}
			fixtureAssert := strings.TrimSpace(entry.Assert)
			count++
			t.Run(fixtureName, func(t *testing.T) {
				if fixtureAssert == "skip" {
					t.Skip("fixture marked as skip")
				}
				input := normalizeFixture(entry.Input)
				expect := normalizeFixture(entry.Expect)
				actual, err := TextileToHtmlWithOptions(input, fixtureOptions)
				if err != nil {
					t.Fatalf("render: %v", err)
				}
				if expect != actual {
					t.Errorf("unexpected output\n-- expected --\n%s\n-- actual --\n%s", expect, actual)
				}
			})
		}
	}
}

func optionsFromSetup(setup []map[string]interface{}) Options {
	options := DefaultOptions()
	for _, item := range setup {
		for key, value := range item {
			switch key {
			case "setLite":
				options.Lite = toBool(value)
			case "setRestricted":
				options.Restricted = toBool(value)
			case "setImages":
				options.Images = toBool(value)
			case "setDimensionlessImages":
				options.DimensionlessImages = toBool(value)
			case "setLinkRelationShip":
				options.LinkRelationship = toString(value)
			case "setLinkPrefix":
				options.LinkPrefix = toString(value)
			case "setImagePrefix":
				options.ImagePrefix = toString(value)
			case "setLineWrap":
				options.LineWrap = toInt(value)
			case "setRawBlocks":
				options.RawBlocks = toBool(value)
			case "setBlockTags":
				options.BlockTags = toBool(value)
			case "class":
				className := toString(value)
				if strings.Contains(className, "DisableSymbols") {
					options.NoGlyphs = true
				}
			}
		}
	}
	return options
}

func toBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true"
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if v == "" {
			return 0
		}
		var parsed int
		_, _ = fmt.Sscanf(v, "%d", &parsed)
		return parsed
	default:
		return 0
	}
}

func normalizeFixture(text string) string {
	return strings.ReplaceAll(text, "\\x20", " ")
}
