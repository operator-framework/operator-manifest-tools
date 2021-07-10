package cmd

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	authFile string
)

// resolveCmd represents the resolve command
var resolveCmd = &cobra.Command{
	Use:     "resolve [flags] IMAGES_FILE",
	Short:   "Resolve a list of image references into their corresponding image reference digests.",
	PreRun:  initOutputWriter,
	PostRun: closeOutputWriter,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(authFile)

		if err != nil {
			log.Fatal(err)
		}

		references := []string{}
		err = json.Unmarshal(data, &references)

		if err != nil {
			log.Fatal(err)
		}

		for i := range references {
			ref := references[i]
			if strings.Contains(ref, "@") {
				continue
			}

		}
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)

	addOutputFlag(resolveCmd)

	resolveCmd.PersistentFlags().StringVarP(&authFile,
		"authfile", "", "The path to the authentication file for registry communication.")
}
