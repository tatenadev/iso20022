package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestXSDFileName(t *testing.T) {
	for in, want := range map[string]string{
		"camt.053.001.14.xsd":        "camt.053.001.14.xsd",
		"camt.053.001.14_0.xsd":      "camt.053.001.14.xsd",
		"pain.001.001.10_1.xsd":      "pain.001.001.10.xsd",
		"dir/camt.053.001.14_12.xsd": "camt.053.001.14.xsd",
		"../../etc/x.xsd":            "x.xsd",
	} {
		if got := xsdFileName(in); got != want {
			t.Errorf("xsdFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteXSD(t *testing.T) {
	progress = io.Discard
	dir := t.TempDir()
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	// A suffixed entry with no plain file is stored under the plain name.
	if err := writeXSD(dir, "a.001.001.01_0.xsd", []byte("v1\r\n")); err != nil {
		t.Fatal(err)
	}
	if got := read("a.001.001.01.xsd"); got != "v1\r\n" {
		t.Fatalf("got %q", got)
	}
	// A suffixed duplicate (same content, other line endings) is skipped,
	// and one with different content doesn't replace the existing file.
	for _, data := range []string{"v1\n", "v2\n"} {
		if err := writeXSD(dir, "a.001.001.01_1.xsd", []byte(data)); err != nil {
			t.Fatal(err)
		}
		if got := read("a.001.001.01.xsd"); got != "v1\r\n" {
			t.Fatalf("after %q: got %q", data, got)
		}
	}
	// An entry without a suffix replaces the file as before.
	if err := writeXSD(dir, "a.001.001.01.xsd", []byte("v3\n")); err != nil {
		t.Fatal(err)
	}
	if got := read("a.001.001.01.xsd"); got != "v3\n" {
		t.Fatalf("got %q", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("want 1 file, got %d", len(entries))
	}
}
