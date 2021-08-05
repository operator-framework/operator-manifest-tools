package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type fileOrCmdParam struct {
	Name         string
	f            *os.File
	outputWriter io.Writer
	outputReader io.Reader
	outputCloser io.Closer
}

func (o *fileOrCmdParam) AddOutputFlag(cmd *cobra.Command, name, defaultVal, description string) {
	cmd.Flags().StringVar(&o.Name, name, defaultVal, description)
}

func (o *fileOrCmdParam) Read(p []byte) (int, error) {
	return o.outputReader.Read(p)
}

func (o *fileOrCmdParam) Write(p []byte) (int, error) {
	return o.outputWriter.Write(p)
}

func (o *fileOrCmdParam) Sync() error {
	if o.f == nil {
		return errors.New("no file present")
	}

	return o.f.Sync()
}

func (o *fileOrCmdParam) Init(cmd *cobra.Command, args []string) error {
	o.outputWriter = cmd.OutOrStdout()
	o.outputReader = cmd.InOrStdin()

	if o.Name != "-" {
		absOutputPath, err := filepath.Abs(o.Name)
		if err != nil {
			return errors.New("failed to get absolute path of file " + o.Name)
		}

		f, err := os.OpenFile(absOutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o755)
		o.outputCloser = f
		o.f = f

		if err != nil {
			return errors.New("failed to open output file " + absOutputPath)
		}

		o.outputWriter = f
		o.outputReader = f
	}

	return nil
}

func (o *fileOrCmdParam) Close() error {
	if o.outputCloser != nil {
		return o.outputCloser.Close()
	}
	return nil
}
