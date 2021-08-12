package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"path/filepath"

	"github.com/operator-framework/operator-manifest-tools/pkg/pullspec"
	"github.com/spf13/cobra"
)

// extractCmdArgs is the args file type for the extractCmd
type extractCmdArgs struct {
	outputFile OutputParam
}

var (
	// extractCmdData stores the data for extractCmd
	extractCmdData = extractCmdArgs{
		outputFile: NewOutputParam(),
	}

	// extractCmd represents the extract command
	extractCmd = &cobra.Command{
		Use:   "extract [flags] MANIFEST_DIR",
		Short: "Identify all the image references in the CSVs found in MANIFEST_DIR.",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return extractCmdData.outputFile.Init(cmd, args)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return extractCmdData.outputFile.Close()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return extract(args[0], &extractCmdData.outputFile)
		},
	}
)

// init initializes the command arguments.
func init() {
	rootCmd.AddCommand(extractCmd)
	extractCmdData.outputFile.AddFlag(extractCmd, "output", "-", `The path to store the extracted image references. Use - to specify stdout. By default - is used.`)
}

// extract will extract images from the CSV located on the path
func extract(manifestPath string, output io.Writer) error {
	out := []interface{}{}

	manifestAbsPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return errors.New("failed to get abs path: " + err.Error())
	}

	log.Printf("extracting image references from %s\n", manifestAbsPath)
	operatorManifests, err := pullspec.FromDirectory(manifestAbsPath, pullspec.DefaultHeuristic)

	for _, manifest := range operatorManifests {
		pullSpecs, err := manifest.GetPullSpecs()
		if err != nil {
			return errors.New("error getting pullspec: " + err.Error())
		}

		for _, pullSpec := range pullSpecs {
			out = append(out, pullSpec.String())
		}
	}

	outBytes, err := json.Marshal(out)
	if err != nil {
		return errors.New("error marshaling json: " + err.Error())
	}

	output.Write(outBytes)
	return nil
}
