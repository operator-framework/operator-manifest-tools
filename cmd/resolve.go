package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
	"github.com/spf13/cobra"
)

type resolveCmdArgs struct {
	input      InputParam
	outputFile OutputParam
	authFile   string
	skopeoPath string
}

var (
	resolveCmdData = &resolveCmdArgs{
		input: NewInputParam(),
		outputFile: NewOutputParam(),
	}
)

// resolveCmd represents the resolve command
var resolveCmd = &cobra.Command{
	Use:   "resolve [flags] IMAGES_FILE",
	Short: "Resolve a list of image references into their corresponding image reference digests. Pass - as an arg if you want to use stdin.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		err := resolveCmdData.outputFile.Init(cmd, args)

		if err != nil {
			return err
		}

		resolveCmdData.input.Name = args[0]
		err = resolveCmdData.input.Init(cmd, args)
		return err
	},
	PostRunE: func(cmd *cobra.Command, args []string) error {
		err1 := resolveCmdData.outputFile.Close()
		err2 := resolveCmdData.input.Close()

		if err1 != nil {
			return err1
		}
		return err2
	},
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return resolve(resolveCmdData.authFile, resolveCmdData.skopeoPath, &resolveCmdData.input, &resolveCmdData.outputFile)
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)

	resolveCmdData.outputFile.AddFlag(resolveCmd, "output", "-", `The path to store the extracted image references. Use - to specify stdout. By default - is used.`)

	resolveCmd.Flags().StringVarP(&resolveCmdData.authFile,
		"authfile", "a", "", "The path to the authentication file for registry communication.")

	resolveCmd.Flags().StringVarP(&resolveCmdData.skopeoPath,
		"skopeo", "s", "skopeo", "The path to skopeo cli utility.")
}

// resolve will read images from the extracted json and write the resolved
// image to the output using skopeo to look up the image shas.
func resolve(
	authfile, skopeoPath string,
	input io.Reader,
	output io.Writer,
) error {
	data, err := io.ReadAll(&resolveCmdData.input)
	if err != nil {
		return errors.New("error reading data: " + err.Error())
	}

	references := []string{}

	err = json.Unmarshal(data, &references)

	if err != nil {
		return errors.New("error unmarshalling references: " + err.Error())
	}

	resolver, err := imageresolver.NewSkopeoResolver(skopeoPath, authfile)

	if err != nil {
		return errors.New("error creating a resolver: " + err.Error())
	}

	results := map[string]string{}
	for i := range references {
		ref := references[i]
		if strings.Contains(ref, "@") {
			continue
		}

		shaRef, err := resolver.ResolveImageReference(ref)
		if err != nil {
			return errors.New("error resolving image: " + err.Error())
		}

		results[ref] = shaRef
	}

	outBytes, err := json.Marshal(results)
	if err != nil {
		return err
	}

	if _, err := output.Write(outBytes); err != nil {
		return errors.New("error writing files: " + err.Error())
	}

	return nil
}
