package cmd

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	// defaultOutputExtract filename
	defaultOutputExtract = "references.json"
	// defaultOutputReplace filename
	defaultOutputReplace = "replacements.json"
)

// pinCmdArgs is the arguments for the command
type pinCmdArgs struct {
	outputExtract OutputParam
	outputReplace OutputParam
	authFile      string
	skopeoPath    string
	dryRun        bool
}

var (
	// pinCmdData is the command 
	pinCmdData = pinCmdArgs{
		outputExtract: NewOutputParam(),
		outputReplace: NewOutputParam(),
	}

	// pinCmd represents the pin command
	pinCmd = &cobra.Command{
		Use:   "pin [flags] MANIFEST_DIR",
		Short: "Pins to digest all the image references from the CSVs found in MANIFEST_DIR.",
		Long: strings.ReplaceAll(`Pins to digest all the image references from the CSVs found in MANIFEST_DIR. For
each image reference, if a tag is used, it is resolved to a digest by querying the
container image registry. Then, replaces all the image references in the CSVs with
the resolved, pinned, version.`, "\n", ""),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := pinCmdData.outputExtract.Init(cmd, args)

			if err != nil {
				return err
			}

			return nil
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pinCmdData.dryRun {
				log.SetOutput(cmd.ErrOrStderr())
			}

			err := extract(args[0], &pinCmdData.outputExtract)

			if err != nil {
				return errors.New("error extracting: " + err.Error())
			}

			pinCmdData.outputExtract.Close()
			inputExtract, err := os.OpenFile(pinCmdData.outputExtract.Name, os.O_RDONLY, 0755)
			if err != nil {
				return errors.New("failure reading extracted data: " + err.Error())
			}
			defer inputExtract.Close()

			if err := pinCmdData.outputReplace.Init(cmd, args); err != nil {
				return errors.New("failure to setup replace output: " + err.Error())
			}

			err = resolve(pinCmdData.authFile,
				pinCmdData.authFile, 
				inputExtract, 
				&pinCmdData.outputReplace)

			if err != nil {
				return errors.New("error resolving: " + err.Error())
			}

			pinCmdData.outputReplace.Close()

			inputReplace, err := os.OpenFile(pinCmdData.outputReplace.Name, os.O_RDONLY, 0755)
			if err != nil {
				return errors.New("failure reading replace data: " + err.Error())
			}
			defer inputReplace.Close()

			err = replace(args[0], inputReplace)

			if err != nil {
				return errors.New("error replacing: " + err.Error())
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(pinCmd)

	pinCmdData.outputExtract.AddFlag(
		pinCmd,
		"output-extract",
		defaultOutputExtract,
		"The path to store the extracted image references from the CSVs. By default "+defaultOutputExtract+" is used.",
	)

	pinCmdData.outputReplace.AddFlag(
		pinCmd,
		"output-replace",
		defaultOutputReplace,
		"The path to store the extracted image reference replacements from the CSVs. By default "+defaultOutputReplace+" is used.",
	)

	pinCmd.Flags().BoolVar(&pinCmdData.dryRun,
		"dry-run", false, strings.ReplaceAll(`When set, replacements are not performed. This is useful to determine if the CSV is
in a state that accepts replacements. 
By default this option is not set.`, "\n", " "))

	pinCmd.Flags().StringVarP(&pinCmdData.authFile,
		"authfile", "a", "", "The path to the authentication file for registry communication.")

	pinCmd.Flags().StringVarP(&pinCmdData.skopeoPath,
		"skopeo", "s", "skopeo", "The path to skopeo cli utility.")
}
