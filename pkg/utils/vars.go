package utils

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// fileOrCmdParam assists in the mounting of output or input variables of a command. It
// can handle a filepath or '-' which indicates to use StdIn and StdOut.
type fileOrCmdParam struct {
	Name   string
	perm   int
	f      *os.File
	writer io.Writer
	reader io.Reader
	closer io.Closer
}

// InputParam is a parameter used for input. If the name is a "-" it will use the commands
// stdin.
type InputParam struct {
	*fileOrCmdParam
}

// OutputParam is a parameter used for output. If the name is a "-" it will use the commands
// stdout.
type OutputParam struct {
	*fileOrCmdParam
}

// NewInputParam creates a new InputParam.
func NewInputParam() InputParam {
	return InputParam{
		fileOrCmdParam: &fileOrCmdParam{perm: os.O_RDONLY},
	}
}

// NewOutputParam creates a new InputParam.
func NewOutputParam() OutputParam {
	return OutputParam{
		fileOrCmdParam: &fileOrCmdParam{perm: os.O_CREATE | os.O_TRUNC | os.O_WRONLY},
	}
}

// AddFlag will add the param to a command
func (o *fileOrCmdParam) AddFlag(cmd *cobra.Command, name, defaultVal, description string) {
	cmd.Flags().StringVar(&o.Name, name, defaultVal, description)
}

// Read reads from the parameters's source
func (o *InputParam) Read(p []byte) (int, error) {
	return o.reader.Read(p)
}

// Write writes to the parameters's source.
func (o *OutputParam) Write(p []byte) (int, error) {
	return o.writer.Write(p)
}

// FromFile initializes the param from a filepath using o.Name
func (o *fileOrCmdParam) FromFile() error {
	absOutputPath, err := filepath.Abs(o.Name)
	if err != nil {
		return errors.New("failed to get absolute path of file " + o.Name)
	}

	f, err := os.OpenFile(absOutputPath, o.perm, 0755)

	if err != nil {
		return errors.New("failed to open output file " + absOutputPath)
	}

	o.closer = f
	o.f = f
	o.writer = f
	o.reader = f
	return nil
}

// Init is a function that sets up the parameter with the intended source.
func (o *fileOrCmdParam) Init(cmd *cobra.Command, args []string) error {
	o.writer = cmd.OutOrStdout()
	o.reader = cmd.InOrStdin()

	if o.Name != "-" {
		return o.FromFile()
	}

	return nil
}

// Close will close the param's input source if applicable.
func (o *fileOrCmdParam) Close() error {
	if o.closer != nil {
		return o.closer.Close()
	}
	return nil
}
