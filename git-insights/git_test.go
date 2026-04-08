package main

import (
	"strings"
	"testing"
)

func TestParseBugClustersOutput(t *testing.T) {
	t.Parallel()

	t.Run("counts and sorts by frequency", func(t *testing.T) {
		t.Parallel()
		var input = "src/auth.go\nsrc/handler.go\nsrc/auth.go\nsrc/auth.go\nsrc/handler.go\nsrc/db.go\n"

		result := parseBugClustersOutput(input)

		if len(result) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(result))
		}
		if result[0].File != "src/auth.go" || result[0].Count != 3 {
			t.Fatalf("expected src/auth.go with count 3, got %s with count %d", result[0].File, result[0].Count)
		}
		if result[1].File != "src/handler.go" || result[1].Count != 2 {
			t.Fatalf("expected src/handler.go with count 2, got %s with count %d", result[1].File, result[1].Count)
		}
		if result[2].File != "src/db.go" || result[2].Count != 1 {
			t.Fatalf("expected src/db.go with count 1, got %s with count %d", result[2].File, result[2].Count)
		}
	})

	t.Run("truncates to top 20", func(t *testing.T) {
		t.Parallel()
		var sb strings.Builder
		for i := range 25 {
			for range 25 - i {
				sb.WriteString("file")
				sb.WriteRune(rune('a' + i))
				sb.WriteString(".go\n")
			}
		}

		result := parseBugClustersOutput(sb.String())

		if len(result) != 20 {
			t.Fatalf("expected 20 entries, got %d", len(result))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := parseBugClustersOutput("")
		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})

	t.Run("skips blank lines", func(t *testing.T) {
		t.Parallel()
		var input = "\n\nfoo.go\n\nbar.go\n\n"

		result := parseBugClustersOutput(input)

		if len(result) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(result))
		}
	})
}

func TestParseActivityOutput(t *testing.T) {
	t.Parallel()

	t.Run("counts and sorts chronologically", func(t *testing.T) {
		t.Parallel()
		var input = "2025-03\n2025-01\n2025-03\n2025-02\n2025-01\n2025-01\n"

		result := parseActivityOutput(input)

		if len(result) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(result))
		}
		if result[0].Month != "2025-01" || result[0].Count != 3 {
			t.Fatalf("expected 2025-01 with count 3, got %s with count %d", result[0].Month, result[0].Count)
		}
		if result[1].Month != "2025-02" || result[1].Count != 1 {
			t.Fatalf("expected 2025-02 with count 1, got %s with count %d", result[1].Month, result[1].Count)
		}
		if result[2].Month != "2025-03" || result[2].Count != 2 {
			t.Fatalf("expected 2025-03 with count 2, got %s with count %d", result[2].Month, result[2].Count)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := parseActivityOutput("")
		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})
}

func TestParseFirefightingOutput(t *testing.T) {
	t.Parallel()

	t.Run("filters matching commits case-insensitively", func(t *testing.T) {
		t.Parallel()
		var input = `abc1234 Add new feature
def5678 Revert "broken auth"
ghi9012 Fix typo in README
jkl3456 HOTFIX: patch production crash
mno7890 Update dependencies
pqr1234 Emergency deploy for login
stu5678 rollback v2.3.1
`

		result := parseFirefightingOutput(input)

		if len(result) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(result))
		}
		if result[0].Hash != "def5678" {
			t.Fatalf("expected first match def5678, got %s", result[0].Hash)
		}
		if result[1].Hash != "jkl3456" {
			t.Fatalf("expected second match jkl3456, got %s", result[1].Hash)
		}
		if result[2].Hash != "pqr1234" {
			t.Fatalf("expected third match pqr1234, got %s", result[2].Hash)
		}
		if result[3].Hash != "stu5678" {
			t.Fatalf("expected fourth match stu5678, got %s", result[3].Hash)
		}
	})

	t.Run("no matching commits", func(t *testing.T) {
		t.Parallel()
		var input = "abc1234 Add feature\ndef5678 Fix typo\n"

		result := parseFirefightingOutput(input)

		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := parseFirefightingOutput("")
		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})
}

func TestParseAuthorsOutput(t *testing.T) {
	t.Parallel()

	t.Run("parses shortlog format", func(t *testing.T) {
		t.Parallel()
		var input = "   150\tAlice Smith\n    42\tBob Jones\n     7\tCharlie\n"

		result := parseAuthorsOutput(input)

		if len(result) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(result))
		}
		if result[0].Name != "Alice Smith" || result[0].Count != 150 {
			t.Fatalf("expected Alice Smith with 150, got %s with %d", result[0].Name, result[0].Count)
		}
		if result[1].Name != "Bob Jones" || result[1].Count != 42 {
			t.Fatalf("expected Bob Jones with 42, got %s with %d", result[1].Name, result[1].Count)
		}
		if result[2].Name != "Charlie" || result[2].Count != 7 {
			t.Fatalf("expected Charlie with 7, got %s with %d", result[2].Name, result[2].Count)
		}
	})

	t.Run("merges duplicate names", func(t *testing.T) {
		t.Parallel()
		var input = "   100\tAlice Smith\n    50\tAlice Smith\n    30\tBob Jones\n    20\tBob Jones\n"

		result := parseAuthorsOutput(input)

		if len(result) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(result))
		}
		if result[0].Name != "Alice Smith" || result[0].Count != 150 {
			t.Fatalf("expected Alice Smith with 150, got %s with %d", result[0].Name, result[0].Count)
		}
		if result[1].Name != "Bob Jones" || result[1].Count != 50 {
			t.Fatalf("expected Bob Jones with 50, got %s with %d", result[1].Name, result[1].Count)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := parseAuthorsOutput("")
		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})
}

func TestParseChurnOutput(t *testing.T) {
	t.Parallel()

	t.Run("counts and sorts by frequency", func(t *testing.T) {
		t.Parallel()
		var input = "main.go\nconfig.go\nmain.go\nmain.go\nconfig.go\nutils.go\n"

		result := parseChurnOutput(input)

		if len(result) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(result))
		}
		if result[0].File != "main.go" || result[0].Count != 3 {
			t.Fatalf("expected main.go with count 3, got %s with count %d", result[0].File, result[0].Count)
		}
		if result[1].File != "config.go" || result[1].Count != 2 {
			t.Fatalf("expected config.go with count 2, got %s with count %d", result[1].File, result[1].Count)
		}
	})

	t.Run("truncates to top 20", func(t *testing.T) {
		t.Parallel()
		var sb strings.Builder
		for i := range 25 {
			for range 25 - i {
				sb.WriteString("file")
				sb.WriteRune(rune('a' + i))
				sb.WriteString(".go\n")
			}
		}

		result := parseChurnOutput(sb.String())

		if len(result) != 20 {
			t.Fatalf("expected 20 entries, got %d", len(result))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := parseChurnOutput("")
		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})
}

func TestParseFrequencyOutput(t *testing.T) {
	t.Parallel()

	t.Run("counts paths from rev-list objects", func(t *testing.T) {
		t.Parallel()
		var input = "abc1234 src/main.go\ndef5678 src/main.go\nghi9012 src/main.go\njkl3456 src/util.go\nmno7890 src/util.go\npqr1234\nstu5678 README.md\n"

		result := parseFrequencyOutput(input)

		if len(result) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(result))
		}
		if result[0].File != "src/main.go" || result[0].Count != 3 {
			t.Fatalf("expected src/main.go with 3, got %s with %d", result[0].File, result[0].Count)
		}
		if result[1].File != "src/util.go" || result[1].Count != 2 {
			t.Fatalf("expected src/util.go with 2, got %s with %d", result[1].File, result[1].Count)
		}
	})

	t.Run("skips lines without paths", func(t *testing.T) {
		t.Parallel()
		var input = "abc1234\ndef5678\nghi9012 only-file.go\n"

		result := parseFrequencyOutput(input)

		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := parseFrequencyOutput("")
		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})
}
