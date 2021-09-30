package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/operator-framework/operator-manifest-tools/cmd/pinning"
	"github.com/spf13/cobra"
)

var (
	Version = "" // set at compile time with -ldflags "-X versserv/cmd.Version=x.y.yz"
	Commit  = ""
	Date    = ""

	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "operator-manifest-tools",
	Short: "",
	Long:  ``,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !verbose {
			log.SetOutput(io.Discard)
		} else {
			log.SetOutput(cmd.ErrOrStderr())
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the tool.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version:", Version)
		fmt.Println("commit:", Commit)
		fmt.Println("date:", Date)
		os.Exit(0)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print debug output of the command")
	rootCmd.AddCommand(pinning.PinningCmd)
	rootCmd.AddCommand(versionCmd)
}

func Root() *cobra.Command {
	return rootCmd
}
