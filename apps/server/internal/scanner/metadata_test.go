package scanner

import "testing"

func TestParseOptionsUnescapesQuotesInAbout(t *testing.T) {
	meta, err := parseOptions([]byte(`define gui.about = _p("""
A \"quoted\" line with a backslash \\ and a newline\nhere.
""")`))
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}
	want := "A \"quoted\" line with a backslash \\ and a newline\nhere."
	if meta.About != want {
		t.Errorf("About = %q, want %q", meta.About, want)
	}
}
