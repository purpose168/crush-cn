package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAttributionMigration 测试归属配置的迁移功能
// 该测试验证了从旧版本的配置格式（co_authored_by 和 generated_with）迁移到新版本格式（trailer_style）的正确性
func TestAttributionMigration(t *testing.T) {
	t.Parallel()

	// 定义测试用例集合，包含各种配置场景
	tests := []struct {
		name             string            // 测试用例名称
		configJSON       string            // 配置JSON字符串
		expectedTrailer  TrailerStyle      // 期望的尾部样式
		expectedGenerate bool              // 期望的生成标记
	}{
		{
			// 测试用例1：旧设置 co_authored_by=true 迁移到 co-authored-by
			name: "旧设置 co_authored_by=true 迁移到 co-authored-by",
			configJSON: `{
				"options": {
					"attribution": {
						"co_authored_by": true,
						"generated_with": false
					}
				}
			}`,
			expectedTrailer:  TrailerStyleCoAuthoredBy,
			expectedGenerate: false,
		},
		{
			// 测试用例2：旧设置 co_authored_by=false 迁移到 none
			name: "旧设置 co_authored_by=false 迁移到 none",
			configJSON: `{
				"options": {
					"attribution": {
						"co_authored_by": false,
						"generated_with": true
					}
				}
			}`,
			expectedTrailer:  TrailerStyleNone,
			expectedGenerate: true,
		},
		{
			// 测试用例3：新设置优先于旧设置
			name: "新设置优先于旧设置",
			configJSON: `{
				"options": {
					"attribution": {
						"trailer_style": "assisted-by",
						"co_authored_by": true,
						"generated_with": false
					}
				}
			}`,
			expectedTrailer:  TrailerStyleAssistedBy,
			expectedGenerate: false,
		},
		{
			// 测试用例4：当两个设置都不存在时的默认值
			name: "当两个设置都不存在时的默认值",
			configJSON: `{
				"options": {
					"attribution": {
						"generated_with": true
					}
				}
			}`,
			expectedTrailer:  TrailerStyleAssistedBy,
			expectedGenerate: true,
		},
		{
			// 测试用例5：当归属设置为 null 时的默认值
			name: "当归属设置为 null 时的默认值",
			configJSON: `{
				"options": {}
			}`,
			expectedTrailer:  TrailerStyleAssistedBy,
			expectedGenerate: true,
		},
	}

	// 遍历所有测试用例并执行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 从JSON字节加载配置
			cfg, err := loadFromBytes([][]byte{[]byte(tt.configJSON)})
			require.NoError(t, err)

			// 设置配置的默认值
			cfg.setDefaults(t.TempDir(), "")

			// 验证尾部样式是否符合预期
			require.Equal(t, tt.expectedTrailer, cfg.Options.Attribution.TrailerStyle)
			// 验证生成标记是否符合预期
			require.Equal(t, tt.expectedGenerate, cfg.Options.Attribution.GeneratedWith)
		})
	}
}
