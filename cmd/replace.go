package cmd

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-manifest-tools/pkg/imagename"
	"github.com/operator-framework/operator-manifest-tools/pkg/pullspec"
	"github.com/spf13/cobra"
)

type replaceCmdArgs struct {
	replacementFile string
	dryRun          bool
}

var (
	replaceCmdData = replaceCmdArgs{}
)

// replaceCmd represents the replace command
var replaceCmd = &cobra.Command{
	Use:   "replace [flags] MANIFEST_DIR",
	Short: "Modify the image references in the CSVs found in the MANIFEST_DIR based on the given REPLACEMENTS_FILE.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		manifestDir := args[0]
		manifestAbsPath, err := filepath.Abs(manifestDir)
		if replaceCmdData.dryRun {
			log.SetOutput(cmd.ErrOrStderr())
		}

		if err != nil {
			log.Fatal(err, " failed to get abs path")
		}

		var replacementsDataIn io.Reader
		if replaceCmdData.replacementFile == "-" {
			replacementsDataIn = cmd.InOrStdin()
		} else {
			replaceFileAbsPath, err := filepath.Abs(replaceCmdData.replacementFile)
			if err != nil {
				log.Fatal(err, " failed to get abs path")
			}

			fileIn, err := os.Open(replaceFileAbsPath)
			if err != nil {
				log.Fatal(err, " failed to open replacements file")
			}
			defer fileIn.Close()

			replacementsDataIn = fileIn
		}

		replacementsData, err := io.ReadAll(replacementsDataIn)
		if err != nil {
			log.Fatal(err, " failed to read data")
		}

		var replacements map[string]string

		err = json.Unmarshal(replacementsData, &replacements)

		if err != nil {
			log.Fatal(err, " failed to convert to json")
		}

		replacementImages := map[imagename.ImageName]imagename.ImageName{}

		for k, v := range replacements {
			key := imagename.Parse(k)
			value := imagename.Parse(v)

			if key == nil || value == nil {
				log.Fatal("failed to parse replacement images")
			}
			replacementImages[*key] = *value
		}

		operatorManifests, err := pullspec.FromDirectory(manifestAbsPath, pullspec.DefaultPullspecHeuristic)

		var output io.Writer
		if outputFile != "" {
			if outputFile == "-" {
				output = cmd.OutOrStdout()
			} else {
				absOutputPath, err := filepath.Abs(outputFile)

				if err != nil {
					log.Fatal(err, " failed to get absolute path")
				}

				outputFile, err := os.OpenFile(absOutputPath,  os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)

				if err != nil {
					log.Fatal(err, " opening file")
				}

				output = outputFile
			}
		}

		for _, manifest := range operatorManifests {
			err := manifest.ReplacePullSpecsEverywhere(replacementImages)

			if err != nil {
				log.Fatal(err, " failed to replace everywhere")
			}

			manifest.SetRelatedImages()

			if replaceCmdData.dryRun {
				log.Println("dryRun is enabled, no output was generated")
				continue
			}

			err = manifest.Dump(output)
			if err != nil {
				log.Fatal(err, " failed to dump the file")
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(replaceCmd)

	replaceCmd.Flags().StringVar(&replaceCmdData.replacementFile,
		"replacements_file", "-", strings.ReplaceAll(`The path to the REPLACEMENTS_FILE. The format of this file is a simple JSON object
where each attribute is a string representing the original image reference and the
value is a string representing the new value for the image reference. Use - to
specify stdin.`, "\n", " "))

	replaceCmd.Flags().BoolVar(&replaceCmdData.dryRun,
		"dry-run", false, strings.ReplaceAll(`When set, replacements are not performed. This is useful to determine if the CSV is
in a state that accepts replacements. By default this option is not set.`, "\n", " "))

	addOutputFlag(replaceCmd)
}
