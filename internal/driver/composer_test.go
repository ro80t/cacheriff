package driver

import "testing"

func TestParseComposerShowOutput(t *testing.T) {
	// Real `composer show --format=json` output with one package.
	out := []byte(`{
    "installed": [
        {
            "name": "psr/log",
            "direct-dependency": true,
            "homepage": "https://github.com/php-fig/log",
            "source": "https://github.com/php-fig/log/tree/3.0.2",
            "version": "3.0.2",
            "description": "Common interface for logging libraries",
            "abandoned": false
        }
    ]
}`)
	parsed, err := parseComposerShowOutput(out)
	if err != nil {
		t.Fatalf("parseComposerShowOutput: %v", err)
	}
	if len(parsed.Installed) != 1 || parsed.Installed[0].Name != "psr/log" || parsed.Installed[0].Version != "3.0.2" {
		t.Errorf("got %+v, want one psr/log 3.0.2 entry", parsed.Installed)
	}
}

func TestParseComposerShowOutputEmpty(t *testing.T) {
	// Real quirk: composer prints a bare "[]" instead of
	// {"installed": []} when there are zero packages.
	parsed, err := parseComposerShowOutput([]byte("[]\n"))
	if err != nil {
		t.Fatalf("parseComposerShowOutput: %v", err)
	}
	if len(parsed.Installed) != 0 {
		t.Errorf("got %+v, want no entries", parsed.Installed)
	}
}
