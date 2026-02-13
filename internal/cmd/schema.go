package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:    "schema",
	Short:  "生成配置文件的 JSON schema",
	Long:   "为 crush 配置文件生成 JSON schema",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		reflector := new(jsonschema.Reflector)
		bts, err := json.MarshalIndent(reflector.Reflect(&config.Config{}), "", "  ")
		if err != nil {
			return fmt.Errorf("无法序列化 schema: %w", err)
		}
		fmt.Println(string(bts))
		return nil
	},
}
