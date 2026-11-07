package cmd

import (
	"github.com/spf13/cobra"

	"github.com/teghnet/iso20022/com"
)

func newCrawlCmd(c *Cli) *cobra.Command {
	return &cobra.Command{
		Use:                   "crawl [dir]",
		Short:                 "Download ISO 20022 XSD files from iso20022.org into dir",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		RunE: c.wrap(func(_ *Cli, _ *cobra.Command, args []string) error {
			return com.Crawl(args)
		}),
	}
}

func newGenerateCmd(c *Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "generate -i <xsd dir> -o <output dir>",
		Short: "Generate Go document packages from ISO 20022 XSD files",
		// com.Generate parses its own -i/-o flags.
		DisableFlagParsing: true,
		RunE: c.wrap(func(_ *Cli, _ *cobra.Command, args []string) error {
			return com.Generate(args)
		}),
	}
}
