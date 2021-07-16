package cmd

import (
	"encoding/json"
	"log"
	"path/filepath"

	"github.com/operator-framework/operator-manifest-tools/pkg/pullspec"
	"github.com/spf13/cobra"
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:     "extract [flags] MANIFEST_DIR",
	Short:   "Identify all the image references in the CSVs found in MANIFEST_DIR.",
	Args:    cobra.ExactArgs(1),
	PreRun:  initOutputWriter,
	PostRun: closeOutputWriter,
	Run: func(cmd *cobra.Command, args []string) {
		out := []interface{}{}

		arg := args[0]
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
				out = append(out, pullSpec.String())
			}
		}

		outBytes, err := json.Marshal(out)
		if err != nil {
			log.Fatal(err, "error marshaling json")
		}

		outputWriter.Write(outBytes)
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)
	addOutputFlag(extractCmd)
}
