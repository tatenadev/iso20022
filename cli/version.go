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

import "fmt"

const appName = "iso20022tpl"

// Version is the application version. It will be overriden during the build process.
// See https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
var Version = "development"

func newVersionCmd() *command {
	return &command{
		name:  "version",
		short: fmt.Sprint("Print the version number of ", appName),
		run: func(c *Cli, _ []string) error {
			_, err := fmt.Fprintf(c.stdout, "%s %s\n", appName, Version)
			return err
		},
	}
}
