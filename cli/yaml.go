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
	"encoding/json"
	"fmt"

	"go.yaml.in/yaml/v4"

	"github.com/tatenadev/iso20022"
)

func newYamlCmd() *command {
	return &command{
		name:  "yaml",
		args:  "<file>",
		short: "Convert ISO 20022 file to yaml",
		nargs: 1,
		run:   runYaml,
	}
}

func runYaml(c *Cli, args []string) error {
	file := args[0]

	doc, err := iso20022.New(file)
	if err != nil {
		return fmt.Errorf("failed to load iso20022 document from '%s': %s", file, err)
	}

	y, err := toYAML(doc)
	if err != nil {
		return err
	}
	_, err = c.stdout.Write(y)
	return err
}

// toYAML converts v to YAML via its JSON encoding, so keys, field order
// and value formatting match the json command.
func toYAML(v any) ([]byte, error) {
	js, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := yaml.Load(js, &n); err != nil {
		return nil, err
	}
	clearStyle(&n)
	return yaml.Dump(&n, yaml.WithIndent(2), yaml.WithQuotePreference(yaml.QuoteDouble))
}

// clearStyle drops the flow and quoting styles the JSON input left on the
// nodes, so they are written in block style and quoted only when needed.
func clearStyle(n *yaml.Node) {
	n.Style = 0
	for _, c := range n.Content {
		clearStyle(c)
	}
}
