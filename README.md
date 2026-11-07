# ISO 20022 Templater

Commandline tool to render templates based on [ISO 20022](https://www.iso20022.org/iso-20022-message-definitions) bank statements.

## Modifications
> by @teghnet

I've reorganized the repo to allow importing.

```text
iso20022
├── cmd - `main` package 
├── com - ready to use commands
├── doc - generated document structs
└── xsd - xsd files (sourced from iso20022.org)
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
