package cmd

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-manifest-tools/pkg/pullspec"
	"github.com/spf13/cobra"
)

var (
	outputFile string
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:        "extract",
	Short:      "Identify all the image references in the CSVs found in MANIFEST_DIR.",
	Args:       cobra.ExactArgs(1),
	ArgAliases: []string{"MANIFEST_DIR"},
	Run: func(cmd *cobra.Command, args []string) {
		outputWriter := cmd.OutOrStdout()

		if outputFile != "-" {
			f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
			defer f.Close()

			if err != nil {
				log.Fatal(err)
			}

			outputWriter = f
		}

		out := []interface{}{}

		for _, arg := range args {
			manifestAbsPath, err := filepath.Abs(arg)
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("extracting image references from %s\n", manifestAbsPath)
			operatorManifests, err := pullspec.FromDirectory(manifestAbsPath, pullspec.DefaultPullspecHeuristic)

			for _, manifest := range operatorManifests {
				pullSpecs, err := manifest.GetPullSpecs()
				if err != nil {
					log.Fatal(err)
				}

				for _, pullSpec := range pullSpecs {
					out = append(out, pullSpec)
				}
			}
		}

		outBytes, err := json.Marshal(out)
		if err != nil {
			log.Fatal(err)
		}

		outputWriter.Write(outBytes)
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)

	// extractCmd.PersistentFlags().StringVar(&manifestDir, "manifest_dir", "",
	//	"The path to the directory containing the manifest files.")
	extractCmd.PersistentFlags().StringVar(&outputFile, "output", "-",
		`The path to store the extracted image references. Use - to specify stdout. By default - is used.`)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// extractCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// extractCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
