package cli

import "testing"

func TestToYAML(t *testing.T) {
	type amt struct {
		Ccy   string
		Value float64
	}
	doc := struct {
		Zeta  string
		Alpha []amt
		Empty string
		Num   string
	}{Zeta: `a: "b"`, Alpha: []amt{{"EUR", 1234.5}}, Num: "0123"}

	got, err := toYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Field order is kept; strings are quoted only when needed.
	want := `Zeta: "a: \"b\""
Alpha:
- Ccy: EUR
  Value: 1234.5
Empty: ""
Num: "0123"
`
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestValidateYAML(t *testing.T) {
	if err := validate([]byte("a:\n  - 1\n"), "yaml", ""); err != nil {
		t.Errorf("valid yaml: %v", err)
	}
	if err := validate([]byte("a: [1, 2\n"), "yaml", ""); err == nil {
		t.Error("invalid yaml: got nil error")
	}
}
