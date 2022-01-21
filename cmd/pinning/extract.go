package pinning

import (
	"encoding/json"
	"errors"
	"io"
	"log"

	"github.com/operator-framework/operator-manifest-tools/internal/utils"
	"github.com/operator-framework/operator-manifest-tools/pkg/image"
	"github.com/operator-framework/operator-manifest-tools/pkg/pullspec"
	"github.com/spf13/cobra"
)

// extractCmdArgs is the args file type for the extractCmd
type extractCmdArgs struct {
	outputFile utils.OutputParam
}

var (
	// extractCmdData stores the data for extractCmd
	extractCmdData = extractCmdArgs{
		outputFile: utils.NewOutputParam(),
	}

	// extractCmd represents the extract command
	extractCmd = &cobra.Command{
		Use: "extract [flags] MANIFEST_DIR",
		Short: `Identify all the image references in the CSVs found
in MANIFEST_DIR.`,
		Args: cobra.ExactArgs(1),
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
	extractCmdData.outputFile.AddFlag(extractCmd, "output", "-",
		`The path to store the extracted image references. Use - to
specify stdout. By default - is used.`)
}

// extract will extract images from the CSV located on the path
func extract(manifestPath string, output io.Writer) error {
	log.Printf("extracting image references from %s\n", manifestPath)
	operatorManifests, err := pullspec.FromDirectory(manifestPath, pullspec.DefaultHeuristic)
	if err != nil {
		return err
	}
	imageNames, err := image.Extract(operatorManifests)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(output).Encode(imageNames); err != nil {
		return errors.New("error marshaling json: " + err.Error())
	}

	return nil
}
