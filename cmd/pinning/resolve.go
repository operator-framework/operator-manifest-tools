package pinning

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/operator-framework/operator-manifest-tools/internal/utils"
	"github.com/operator-framework/operator-manifest-tools/pkg/image"
	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
	"github.com/spf13/cobra"
)

type resolveCmdArgs struct {
	resolver     string
	resolverArgs map[string]string
	authFile     string

	input      utils.InputParam
	outputFile utils.OutputParam
}

var (
	resolveCmdData = &resolveCmdArgs{
		input:      utils.NewInputParam(true),
		outputFile: utils.NewOutputParam(),
	}
)

// resolveCmd represents the resolve command
var resolveCmd = &cobra.Command{
	Use:   "resolve [flags] IMAGES_FILE",
	Short: "Resolve a list of image tas to shas.",
	Long:  `Resolve a list of image references into their corresponding image reference digests. Pass - as an arg if you want to use stdin.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		resolveCmdData.input.Name = args[0]
		err := resolveCmdData.input.Init(cmd, args)

		if err != nil {
			return err
		}

		err = resolveCmdData.outputFile.Init(cmd, args)

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
		resolverArgs := resolveCmdData.resolverArgs

		if resolverArgs == nil {
			resolverArgs = make(map[string]string)
		}

		if file := resolveCmdData.authFile; file != "" {
			resolverArgs["authFile"] = file
		}

		resolver, err := imageresolver.GetResolver(
			imageresolver.ResolverOption(resolveCmdData.resolver), resolverArgs)

		if err != nil {
			return fmt.Errorf("failed to get a resolver: %s", err)
		}

		return resolve(
			resolver,
			&resolveCmdData.input,
			&resolveCmdData.outputFile,
		)
	},
}

func init() {
	resolveCmdData.outputFile.AddFlag(resolveCmd, "output", "-", `The path to store the extracted image references. Use - to specify stdout. By default - is used.`)

	// legacy support flag
	resolveCmd.Flags().StringVarP(&resolveCmdData.authFile,
		"authfile", "a", "", `The path to the authentication file for registry
communication using skopeo. Uses skopeo's default if not provided.`)

	mountResolverOpts(resolveCmd, &resolveCmdData.resolver, &resolveCmdData.resolverArgs)
}

var runSkopeoLocationCmd sync.Once
var skopeoLocation = ""

func getSkopeoLocation() {
	skopeoWhich, err := exec.Command("command", "-v", "skopeo").Output()

	if err == nil {
		skopeoLocation = strings.TrimSpace(string(skopeoWhich))
	}
}

func mountResolverOpts(
	cmd *cobra.Command,
	resolverVar *string,
	resolverArgs *map[string]string,
) {
	runSkopeoLocationCmd.Do(getSkopeoLocation)

	var resolverDefaults map[string]string

	if skopeoLocation != "" {
		resolverDefaults = map[string]string{"path": skopeoLocation}
	}

	cmd.Flags().StringVarP(resolverVar,
		"resolver", "r", "skopeo",
		fmt.Sprintf("The resolver to use; valid values are [%s]", imageresolver.GetResolverOptions()))

	cmd.Flags().StringToStringVar(resolverArgs,
		"resolver-args",
		resolverDefaults,
		"The resolver to use; valid values are skopeo or script")
}

// resolve will read images from the extracted json and write the resolved
// image to the output using skopeo to look up the image shas.
func resolve(
	resolver imageresolver.ImageResolver,
	input io.Reader,
	output io.Writer,
) error {
	references := []string{}
	if err := json.NewDecoder(input).Decode(&references); err != nil {
		return errors.New("error unmarshalling references: " + err.Error())
	}
	replacements, err := image.Resolve(resolver, references)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(output).Encode(replacements); err != nil {
		return errors.New("error writing files: " + err.Error())
	}

	return nil
}
