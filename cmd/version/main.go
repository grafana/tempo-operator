package version

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/os-observability/tempo-operator/internal/version"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the version of the Tempo Operator and exit",
		RunE: func(cmd *cobra.Command, args []string) error {
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
