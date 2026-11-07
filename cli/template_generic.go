/*
Copyright © 2021 Michael Gruener

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"text/template"
	"unicode/utf8"

	"sigs.k8s.io/yaml"

	"github.com/tatenadev/iso20022"
)

type templateOpts struct {
	output       string
	validate     string
	csvSeparator string
}

func newTemplateGenericCmd() *command {
	var o templateOpts
	return &command{
		name:  "generic",
		args:  "<iso20022 file> <template>",
		short: "Render a generic template based on a ISO 20022 file",
		nargs: 2,
		flags: func(fs *flagSet) {
			fs.string(&o.output, "output", "o", "", "Write output to file")
			fs.string(&o.validate, "validate", "", "", "Validate the result as (json,yaml,csv)")
			fs.string(&o.csvSeparator, "csv-separator", "", ",", "CSV separator used when validating csv")
		},
		run: func(c *Cli, args []string) error {
			return runTemplateGeneric(c.stdout, args[0], args[1], o)
		},
	}
}

func runTemplateGeneric(w io.Writer, isoFile, templateFile string, o templateOpts) error {
	doc, err := iso20022.New(isoFile)
	if err != nil {
		return fmt.Errorf("failed to load iso20022 document from '%s': %s", isoFile, err)
	}

	td, err := template.ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to load template from '%s': %s", templateFile, err)
	}

	var buf bytes.Buffer
	if err := td.Execute(&buf, doc); err != nil {
		return fmt.Errorf("failed to render template: %s", err)
	}

	if err := validate(buf.Bytes(), o.validate, o.csvSeparator); err != nil {
		return fmt.Errorf("failed to validate result as '%s': %s", o.validate, err)
	}

	return output(w, buf.Bytes(), o.output)
}

func validate(data []byte, as, csvSeparator string) error {
	var doc interface{}
	switch as {
	case "":
		return nil
	case "json":
		return json.Unmarshal(data, &doc)
	case "yaml":
		return yaml.Unmarshal(data, &doc)
	case "csv":
		return validateCSV(data, csvSeparator)
	default:
		return fmt.Errorf("unknown validation format: %s", as)
	}
}

func validateCSV(data []byte, sep string) error {
	comma, size := utf8.DecodeRuneInString(sep)
	if size == 0 || size != len(sep) {
		return fmt.Errorf("csv separator must be a single character, got %q", sep)
	}
	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.Comma = comma

	for {
		if _, err := csvReader.Read(); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func output(w io.Writer, data []byte, dest string) error {
	if dest != "" {
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("failed to write output to '%s': %s", dest, err)
		}
		return nil
	}

	_, err := w.Write(data)
	return err
}
