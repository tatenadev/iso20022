# ISO 20022 Templater

Commandline tool to render templates based on [ISO 20022](https://www.iso20022.org/iso-20022-message-definitions) bank statements.

## Modifications
> by @teghnet

I've reorganized the repo to allow importing.

```text
iso20022
├── cmd/iso20022tpl - `main` package of the CLI
├── com - CLI commands (importable)
├── doc - generated document structs
└── xsd - xsd files (sourced from iso20022.org)
```

Install the CLI with:

```sh
go install github.com/teghnet/iso20022/cmd/iso20022tpl@latest
```

## Usage

```text
iso20022tpl [global flags] <command> [args]
```

### Global flags

| Flag | Default | Description |
|---|---|---|
| `--log-level` | `warning` | Log level: `trace`, `debug`, `info`, `warn`/`warning`, `error`, `fatal`, `panic` |
| `--json-log` | `false` | Write log messages as JSON |
| `--log-functions` | `false` | Include function names in log output (performance impact) |
| `-q`, `--quiet` | `false` | Hide progress messages from `crawl` and `generate`; errors and command results are still printed |

Every flag can also be set by an environment variable: prefix `ISO20022TPL_`, upper case, with `-` replaced by `_`. For example, `ISO20022TPL_LOG_LEVEL=debug`.

### `json <file>`

Load an ISO 20022 XML document and print it as indented JSON.

```sh
iso20022tpl json statement.xml
```

### `yaml <file>`

Load an ISO 20022 XML document and print it as YAML.

```sh
iso20022tpl yaml statement.xml
```

### `template generic <file> <template>`

Render a Go [`text/template`](https://pkg.go.dev/text/template) with the loaded ISO 20022 document as its data (`.`). The field names are the ones shown by `json`/`yaml`.

| Flag | Default | Description |
|---|---|---|
| `-o`, `--output` | stdout | Write the result to this file |
| `--validate` | none | Check that the result parses as `json`, `yaml` or `csv` |
| `--csv-separator` | `,` | Separator used by `--validate csv` |

```sh
iso20022tpl template generic --validate csv -o statement.csv statement.xml firefly.csv.tpl
```

### `crawl <dir>`

Download every message definition archive (current catalogue and archive) from [iso20022.org](https://www.iso20022.org/iso-20022-message-definitions) and extract all `.xsd` files into `<dir>`. Nested zip files are unpacked too. HTTP responses are cached in `./cache`.

```sh
iso20022tpl crawl xsd
```

### `generate -i <xsd dir> -o <output dir>`

Generate a Go package for each `*.xsd` file in `<xsd dir>`, plus `<output dir>/iso20022.gen.go`, which maps each schema namespace to its package so `json`, `yaml` and `template` can load any supported document. `head.*` schemas (business application headers) are skipped.

Packages are written to `<output dir>/document/<area>/<message>/<variant>/<version>/`. `iso20022.gen.go` imports them as `github.com/teghnet/iso20022/<output dir>/document/...`, so run the command from the repository root with a relative `-o` path.

```sh
iso20022tpl generate -i xsd -o .
```

### `version`

Print the version of `iso20022tpl`.

### `completion <shell>`

Generate a shell completion script for `bash`, `zsh`, `fish` or `powershell`.

```sh
source <(iso20022tpl completion bash)
```

## Filtering documents

ISO20022 documents can be huge and, depending on the bank, contain mostly empty fields. The following command can be used to get a "clean" version of a document to check which fields can be used for templating:

```sh
iso20022tpl json <document> | jq 'walk( if type == "object" then with_entries(select(
      (.value != null) and
      (.value != "") and
      (.value != {}) and
      (.value != []) and
      (.value != "0001-01-01T00:00:00Z") and
      (.value != "0001-01-01Z") and
      (.value != "0001-01Z") and
      (.value != "NOTPROVIDED") and
      (.value != 0)
   )) else . end)'
```

The different `(.value != ...)` statements might have to be changed depending on what values can be considered "empty" for the given document.
