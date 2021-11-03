package pinning

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/operator-framework/operator-manifest-tools/internal/utils"
	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
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
	resolver     string
	resolverArgs map[string]string
	authFile     string
	dryRun       bool

	outputExtract utils.OutputParam
	outputReplace utils.OutputParam
}

var (
	// pinCmdData is the command
	pinCmdData = pinCmdArgs{
		outputExtract: utils.NewOutputParam(),
		outputReplace: utils.NewOutputParam(),
	}

	// pinCmd represents the pin command
	pinCmd = &cobra.Command{
		Use:   "pin [flags] MANIFEST_DIR",
		Short: "Pins to digest all the image references from the CSVs found in MANIFEST_DIR.",
		Long: `Pins to digest all the image references from the CSVs found in MANIFEST_DIR. For
each image reference, if a tag is used, it is resolved to a digest by querying the
container image registry. Then, replaces all the image references in the CSVs with
the resolved, pinned, version.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := utils.CheckIfDirectoryExists(args[0]); err != nil {
				return err
			}

			if err := pinCmdData.outputExtract.Init(cmd, args); err != nil {
				return err
			}

			return nil
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pinCmdData.dryRun {
				log.SetOutput(cmd.ErrOrStderr())
			}

			manifestDir := args[0]
			resolverArgs := pinCmdData.resolverArgs

			if resolverArgs == nil {
				resolverArgs = make(map[string]string)
			}

			resolverArgs["authFile"] = pinCmdData.authFile

			resolver, err := imageresolver.GetResolver(
				imageresolver.ResolverOption(pinCmdData.resolver), resolverArgs)

			if err != nil {
				return fmt.Errorf("failed to get a resolver: %s", err)
			}

			return pin(
				manifestDir,
				resolver,
				pinCmdData.outputExtract,
				pinCmdData.outputReplace,
			)
		},
	}
)

func pin(
	manifestDir string,
	resolver imageresolver.ImageResolver,
	outputExtract, outputReplace utils.OutputParam,
) error {
	err := outputExtract.FromFile()
	if err != nil {
		return errors.New("error extracting: " + err.Error())
	}

	err = extract(manifestDir, &outputExtract)

	if err != nil {
		return errors.New("error extracting: " + err.Error())
	}

	outputExtract.Close()
	inputExtract, err := os.OpenFile(outputExtract.Name, os.O_RDONLY, 0755)
	if err != nil {
		return errors.New("failure reading extracted data: " + err.Error())
	}
	defer inputExtract.Close()

	if err := outputReplace.FromFile(); err != nil {
		return errors.New("failure to setup replace output: " + err.Error())
	}

	err = resolve(resolver,
		inputExtract,
		&outputReplace)

	if err != nil {
		return errors.New("error resolving: " + err.Error())
	}

	outputReplace.Close()

	inputReplace, err := os.OpenFile(outputReplace.Name, os.O_RDONLY, 0755)
	if err != nil {
		return errors.New("failure reading replace data: " + err.Error())
	}
	defer inputReplace.Close()

	err = replace(manifestDir, inputReplace)

	if err != nil {
		return errors.New("error replacing: " + err.Error())
	}

	return nil
}

func init() {
	pinCmdData.outputExtract.AddFlag(
		pinCmd,
		"output-extract",
		defaultOutputExtract,
		`The path to store the extracted image references from the CSVs.
By default `+defaultOutputExtract+" is used.",
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

	mountResolverOpts(pinCmd, &pinCmdData.resolver, &pinCmdData.resolverArgs)
}
