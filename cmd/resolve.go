package cmd

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
	"github.com/spf13/cobra"
)

type resolveCmdArgs struct {
	authFile   string
	skopeoPath string
}

var (
	resolveCmdData = &resolveCmdArgs{}
)

// resolveCmd represents the resolve command
var resolveCmd = &cobra.Command{
	Use:     "resolve [flags] IMAGES_FILE",
	Short:   "Resolve a list of image references into their corresponding image reference digests. Pass - as an arg if you want to use stdin.",
	PreRun:  initOutputWriter,
	PostRun: closeOutputWriter,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var dataIn io.Reader

		firstArg := args[0]

		if firstArg == "-" {
			dataIn = cmd.InOrStdin()
		} else {
			manifestAbsPath, err := filepath.Abs(args[0])
			if err != nil {
				log.Fatal(err, " failed to get abs path")
			}

			fileIn, err := os.Open(manifestAbsPath)
			if err != nil {
				log.Fatal(err, " failed to open file")
			}

			defer fileIn.Close()

			dataIn = fileIn
		}

		data, err := io.ReadAll(dataIn)
		if err != nil {
			log.Fatal(err, " failed to read data")
		}

		references := []string{}

		err = json.Unmarshal(data, &references)

		if err != nil {
			log.Fatal(err)
		}

		resolver, err := imageresolver.NewSkopeoImageResolver(resolveCmdData.skopeoPath, resolveCmdData.authFile)

		if err != nil {
			log.Fatal(err)
		}

		results := map[string]string{}
		for i := range references {
			ref := references[i]
			if strings.Contains(ref, "@") {
				continue
			}

			shaRef, err := resolver.ResolveImageReference(ref)
			if err != nil {
				log.Fatal(err, " error resolving image")
			}

			results[ref] = shaRef
		}

		outBytes, err := json.Marshal(results)
		if err != nil {
			log.Fatal(err)
		}

		outputWriter.Write(outBytes)
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)

	addOutputFlag(resolveCmd)

	resolveCmd.Flags().StringVarP(&resolveCmdData.authFile,
		"authfile", "a", "", "The path to the authentication file for registry communication.")

	resolveCmd.Flags().StringVarP(&resolveCmdData.skopeoPath,
		"skopeo", "s", "skopeo", "The path to skopeo cli utility.")
}
