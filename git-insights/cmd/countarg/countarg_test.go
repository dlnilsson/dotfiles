package countarg

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	if n, err := Parse(nil, 7); err != nil || n != 7 {
		t.Errorf("Parse(nil) = %d, %v; want 7", n, err)
	}
	if n, err := Parse([]string{"30"}, 7); err != nil || n != 30 {
		t.Errorf("Parse(30) = %d, %v; want 30", n, err)
	}
	for _, bad := range []string{"0", "-1", "abc", ""} {
		if _, err := Parse([]string{bad}, 7); err == nil {
			t.Errorf("Parse(%q): expected error", bad)
		}
	}
}
