package cmd

import (
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFile   string
	outputWriter io.Writer
	outputCloser  io.Closer
)

func addOutputFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&outputFile, "output", "-",
		`The path to store the extracted image references. Use - to specify stdout. By default - is used.`)
}

func initOutputWriter(cmd *cobra.Command, args []string)  {
	outputWriter = cmd.OutOrStdout()

	if outputFile != "-" {
		f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
		outputCloser = f

		if err != nil {
			log.Fatal(err)
		}

		outputWriter = f
	}
}

func closeOutputWriter(cmd *cobra.Command, args []string)  {
	if outputCloser != nil {
		err := outputCloser.Close()

		if err != nil {
			log.Fatal(err)
		}
	}
}
