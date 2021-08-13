package pinning

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
	"github.com/operator-framework/operator-manifest-tools/pkg/utils"
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
		input:      utils.NewInputParam(),
		outputFile: utils.NewOutputParam(),
	}
)

// resolveCmd represents the resolve command
var resolveCmd = &cobra.Command{
	Use:   "resolve [flags] IMAGES_FILE",
	Short: "Resolve a list of image tas to shas.",
	Long: `Resolve a list of image references into their corresponding image reference digests. Pass - as an arg if you want to use stdin.`,
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
var skopeoLocation = "skopeo"

func getSkopeoLocation() {
	skopeoWhich, err := exec.Command("which", "skopeo").Output()

	if err == nil {
		skopeoLocation = strings.TrimSpace(string(skopeoWhich))
	} else {
		log.Printf("which skopeo command failed, skopeo may not be available on the path")
	}
}

func mountResolverOpts(
	cmd *cobra.Command,
	resolverVar *string,
	resolverArgs *map[string]string,
) {
	runSkopeoLocationCmd.Do(getSkopeoLocation)

	cmd.Flags().StringVarP(resolverVar,
		"resolver", "r", "skopeo",
		fmt.Sprintf("The resolver to use; valid values are [%s]", imageresolver.GetResolverOptions()))

	cmd.Flags().StringToStringVar(resolverArgs,
		"resolver-args", map[string]string{"path": skopeoLocation}, "The resolver to use; valid values are skopeo or script")
}

// resolve will read images from the extracted json and write the resolved
// image to the output using skopeo to look up the image shas.
func resolve(
	resolver imageresolver.ImageResolver,
	input io.Reader,
	output io.Writer,
) error {
	data, err := io.ReadAll(input)
	if err != nil {
		return errors.New("error reading data: " + err.Error())
	}

	references := []string{}

	err = json.Unmarshal(data, &references)

	if err != nil {
		return errors.New("error unmarshalling references: " + err.Error())
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
