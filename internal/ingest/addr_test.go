package ingest

import "testing"

func TestNormalizeAddr(t *testing.T) {
	cases := map[string]string{
		"":                         "",
		"off":                      "off",
		"9477":                     "127.0.0.1:9477",
		":9477":                    "127.0.0.1:9477",
		"127.0.0.1:9477":           "127.0.0.1:9477",
		"http://127.0.0.1:9477":    "127.0.0.1:9477",
		"http://127.0.0.1:9477/in": "127.0.0.1:9477",
		"0.0.0.0:9477":             "127.0.0.1:9477",
	}
	for in, want := range cases {
		if got := NormalizeAddr(in); got != want {
			t.Errorf("NormalizeAddr(%q) = %q, want %q", in, got, want)
		}
	}
}
