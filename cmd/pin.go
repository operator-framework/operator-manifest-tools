package cmd

import (
	"errors"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultOutputExtract = "references.json"
	defaultOutputReplace = "replacements.json"
)

type pinCmdArgs struct {
	outputExtract fileOrCmdParam
	outputReplace fileOrCmdParam
	authFile      string
	skopeoPath    string
	dryRun        bool
}

var (
	pinCmdData = pinCmdArgs{}

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

			return pinCmdData.outputReplace.Init(cmd, args)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			err1 := pinCmdData.outputExtract.Close()
			err2 := pinCmdData.outputReplace.Close()

			if err1 != nil {
				return err1
			}

			return err2
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
			
			pinCmdData.outputExtract.Sync()

			err = resolve(pinCmdData.authFile, pinCmdData.authFile, &pinCmdData.outputExtract, &pinCmdData.outputReplace)

			if err != nil {
				return errors.New("error resolving: " + err.Error())
			}

			pinCmdData.outputReplace.Sync()

			err = replace(args[0], &pinCmdData.outputReplace)

			if err != nil {
				return errors.New("error replacing: " + err.Error())
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(pinCmd)

	pinCmdData.outputExtract.AddOutputFlag(
		pinCmd,
		"output-extract",
		defaultOutputExtract,
		"The path to store the extracted image references from the CSVs. By default "+defaultOutputExtract+" is used.",
	)

	pinCmdData.outputReplace.AddOutputFlag(
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
