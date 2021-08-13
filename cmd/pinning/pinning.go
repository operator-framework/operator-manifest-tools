package pinning

import "github.com/spf13/cobra"

var PinningCmd = &cobra.Command{
	Use: "pinning",
	Short: "Operator manifest image pinning",
	Long: "Includes pinning functionality for extracting images from manifest files and replacing (pinning) the image references with image digests.",
}

func init() {
	PinningCmd.AddCommand(pinCmd)
	PinningCmd.AddCommand(replaceCmd)
	PinningCmd.AddCommand(extractCmd)
	PinningCmd.AddCommand(resolveCmd)
}
