package cli

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gocolly/colly/v2"
	"github.com/spf13/cobra"
)

// progress receives informational messages; --quiet sets it to io.Discard.
var progress io.Writer = os.Stdout

func newCrawlCmd(c *Cli) *cobra.Command {
	return &cobra.Command{
		Use:                   "crawl [dir]",
		Short:                 "Download ISO 20022 XSD files from iso20022.org into dir",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: c.wrap(func(_ *Cli, _ *cobra.Command, args []string) error {
			return Crawl(args)
		}),
	}
}

func Crawl(args []string) error {
	dir := args[0]
	err := os.MkdirAll(dir, 0o777)
	if err != nil {
		return err
	}
	fmt.Fprintf(progress, "Writing files to '%s'\n", dir)

	// Instantiate default collector
	c := colly.NewCollector(
		colly.AllowedDomains("www.iso20022.org"),
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"),
		colly.CacheDir(".cache"),
	)

	// Add headers on all requests
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Sec-GPC", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Sec-Fetch-User", "?1")
	})

	// Visit all download links of the current ISO 20022 message catalogue
	c.OnHTML("a[href$=download][aria-label=Download]", func(e *colly.HTMLElement) {
		fmt.Fprintf(progress, "AAA %s\n", e.Attr("href"))
		err := e.Request.Visit(e.Attr("href"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to visit %s: %s\n", e.Attr("href"), err)
		}
	})

	// Visit all download links of the ISO 20022 message catalogue archive
	c.OnHTML("a[href$=download][aria-label='Download Complete Message Set']", func(e *colly.HTMLElement) {
		fmt.Fprintf(progress, "BBB %s\n", e.Attr("href"))
		err := e.Request.Visit(e.Attr("href"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to visit %s: %s\n", e.Attr("href"), err)
		}
	})

	// Handle paging
	c.OnHTML("a[href][title^='Go to page']", func(e *colly.HTMLElement) {
		fmt.Fprintf(progress, "CCC next page\n")
		err := e.Request.Visit(e.Attr("href"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to visit %s: %s\n", e.Attr("href"), err)
		}
	})

	// Download message catalogue zip files
	c.OnResponse(func(r *colly.Response) {
		fmt.Fprintf(progress, "Response received: %s\n", r.Request.URL)
		if strings.Index(r.Headers.Get("Content-Type"), "application/zip") > -1 {
			fmt.Fprintf(progress, "Processing %s\n", r.FileName())
			if err := unzipWriteFile(r.Body, dir); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to unzip file '%s': %s\n", r.FileName(), err)
			}
		}
	})

	err = c.Visit("https://www.iso20022.org/iso-20022-message-definitions?page=0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to visit %s: %s\n", "https://www.iso20022.org/iso-20022-message-definitions?page=0", err)
	}
	err = c.Visit("https://www.iso20022.org/catalogue-messages/iso-20022-messages-archive?page=0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to visit %s: %s\n", "https://www.iso20022.org/catalogue-messages/iso-20022-messages-archive?page=0", err)
	}
	return nil
}

func unzipWriteFile(data []byte, dir string) error {
	mime, err := mimetype.DetectReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to detect mimetype")
	}
	if !mime.Is("application/zip") {
		return fmt.Errorf("data is not a zipfile")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to read zipped data: %s (%s)", err, mime)
	}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		// we are only interested in xml schema and zip files
		if path.Ext(zipFile.Name) != ".xsd" && path.Ext(zipFile.Name) != ".zip" {
			fmt.Fprintf(progress, "ignoring[%s] %s\n", path.Ext(zipFile.Name), zipFile.Name)
			continue
		}
		fmt.Fprintf(progress, "  extracting %s\n", zipFile.Name)
		unzippedFileBytes, err := readZipFile(zipFile)
		if err != nil {
			return fmt.Errorf("failed to unzip file '%s': %s", zipFile.Name, err)
		}

		if path.Ext(zipFile.Name) == ".zip" {
			if err := unzipWriteFile(unzippedFileBytes, dir); err != nil {
				return err
			}
			continue
		}

		if err := writeXSD(dir, zipFile.Name, unzippedFileBytes); err != nil {
			return err
		}
	}

	return nil
}

// uploadSuffix matches the "_0", "_1", ... that iso20022.org adds to the
// names of schemas uploaded more than once (e.g. camt.053.001.14_0.xsd).
var uploadSuffix = regexp.MustCompile(`_[0-9]+(\.xsd)$`)

// xsdFileName returns the name an extracted schema is stored under: the
// base name of the zip entry (so entries can't escape dir) without an
// upload suffix.
func xsdFileName(zipName string) string {
	return uploadSuffix.ReplaceAllString(path.Base(zipName), "$1")
}

// writeXSD stores a schema extracted from a zip entry in dir. A schema whose
// entry name had an upload suffix never replaces an existing file: if the
// contents match it is skipped, otherwise a warning is printed.
func writeXSD(dir, zipName string, data []byte) error {
	name := xsdFileName(zipName)
	target := filepath.Join(dir, name)
	if name != path.Base(zipName) {
		existing, err := os.ReadFile(target)
		switch {
		case err == nil && sameXSD(existing, data):
			fmt.Fprintf(progress, "  skipping %s: same as %s\n", zipName, name)
			return nil
		case err == nil:
			fmt.Fprintf(os.Stderr, "Keeping existing %s: %s has different contents\n", target, zipName)
			return nil
		case !errors.Is(err, fs.ErrNotExist):
			return fmt.Errorf("failed to read '%s': %s", target, err)
		}
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("failed to save extracted file as '%s': %s", target, err)
	}
	return nil
}

// sameXSD reports whether a and b are equal, ignoring CRLF vs LF line endings.
func sameXSD(a, b []byte) bool {
	crlf := []byte("\r\n")
	lf := []byte("\n")
	return bytes.Equal(bytes.ReplaceAll(a, crlf, lf), bytes.ReplaceAll(b, crlf, lf))
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}
