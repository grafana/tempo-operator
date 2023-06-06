package version

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/os-observability/tempo-operator/internal/version"
)

// NewVersionCommand returns a new version command.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the version of the Tempo Operator and exit",
		RunE: func(c *cobra.Command, args []string) error {
			info := version.Get()
			json, err := json.Marshal(info)
			if err != nil {
				return err
			}
			fmt.Println(string(json))
			return nil
		},
	}

	return cmd
}
