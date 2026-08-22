package textwrap

import (
	"strings"
	"testing"
)

func TestContentLineFitsAsIs(t *testing.T) {
	got := ContentLine("short line", 40, 4)
	if len(got) != 1 || got[0] != "short line" {
		t.Errorf("got %v, want [\"short line\"] unchanged", got)
	}
}

func TestContentLineWrapsLongPath(t *testing.T) {
	prefix := "  Git database                              0 B  "
	path := strings.Repeat("a", 60)
	line := prefix + path

	width := len(prefix) + 20
	chunks := ContentLine(line, width, len(prefix))
	if len(chunks) < 2 {
		t.Fatalf("got %d chunk(s), want at least 2 for a line longer than the width: %v", len(chunks), chunks)
	}
	for i, c := range chunks {
		if len([]rune(c)) > width {
			t.Errorf("chunk %d is %d runes wide, want <= %d: %q", i, len([]rune(c)), width, c)
		}
	}
	// Every chunk after the first should be indented to len(prefix).
	for i, c := range chunks[1:] {
		if !strings.HasPrefix(c, strings.Repeat(" ", len(prefix))) {
			t.Errorf("continuation chunk %d not indented by %d spaces: %q", i+1, len(prefix), c)
		}
	}
	// Reassembling (stripping the injected indent) should recover the
	// original text with nothing dropped.
	var rebuilt strings.Builder
	for i, c := range chunks {
		if i == 0 {
			rebuilt.WriteString(c)
			continue
		}
		rebuilt.WriteString(strings.TrimPrefix(c, strings.Repeat(" ", len(prefix))))
	}
	if rebuilt.String() != line {
		t.Errorf("rebuilt text doesn't match original.\ngot:  %q\nwant: %q", rebuilt.String(), line)
	}
}

func TestContentLineNeverExceedsWidthEvenWithBadIndent(t *testing.T) {
	// indent >= width should fall back to no indent rather than loop
	// forever or emit an oversized chunk.
	chunks := ContentLine(strings.Repeat("x", 30), 10, 999)
	if len(chunks) == 0 {
		t.Fatal("got no chunks")
	}
	for _, c := range chunks {
		if len([]rune(c)) > 10 {
			t.Errorf("chunk exceeds width 10: %q", c)
		}
	}
}

func TestContentLineZeroWidth(t *testing.T) {
	got := ContentLine("anything", 0, 0)
	if len(got) != 1 || got[0] != "anything" {
		t.Errorf("got %v, want the line returned unchanged when width<=0", got)
	}
}

func TestContentLineBreaksAfterSeparator(t *testing.T) {
	// A window that would otherwise cut mid-segment ("...tkata\.ca")
	// should instead break right after the last backslash in view.
	line := `  C:\Users\tkata\.cargo\registry\src\index.crates.io-abc123`

	chunks := ContentLine(line, 20, 2)
	if len(chunks) < 2 {
		t.Fatalf("got %d chunk(s), want at least 2: %v", len(chunks), chunks)
	}

	first := chunks[0]
	last := []rune(first)[len([]rune(first))-1]
	if last != '\\' && last != '/' {
		t.Errorf("first chunk should end right after a separator, got %q", first)
	}

	for i, c := range chunks {
		if got := len([]rune(c)); got > 20 {
			t.Errorf("chunk %d is %d runes wide, want <= 20: %q", i, got, c)
		}
	}

	var rebuilt strings.Builder
	for i, c := range chunks {
		if i == 0 {
			rebuilt.WriteString(c)
			continue
		}
		rebuilt.WriteString(strings.TrimPrefix(c, "  "))
	}
	if rebuilt.String() != line {
		t.Errorf("rebuilt text doesn't match original.\ngot:  %q\nwant: %q", rebuilt.String(), line)
	}
}

func TestLastPathSeparator(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"no separators here", 0},
		{`C:\Users\tkata`, 9}, // right after the last backslash
		{"scope/pkg", 6},      // right after the slash
		{`\leading`, 0},       // separator at index 0 doesn't count
	}
	for _, tt := range tests {
		if got := lastPathSeparator([]rune(tt.in)); got != tt.want {
			t.Errorf("lastPathSeparator(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
