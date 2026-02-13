package config

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/catwalk/pkg/embedded"
	"github.com/charmbracelet/x/etag"
	"github.com/purpose168/crush-cn/internal/agent/hyper"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/home"
)

// syncer 定义了同步器接口，用于获取类型为 T 的数据
type syncer[T any] interface {
	Get(context.Context) (T, error)
}

// 全局变量，用于确保提供者列表只加载一次
var (
	providerOnce sync.Once              // 确保提供者列表只初始化一次
	providerList []catwalk.Provider     // 缓存的提供者列表
	providerErr  error                  // 获取提供者时产生的错误
)

// cachePathFor 根据名称生成缓存文件路径
// 支持跨平台：Windows、Linux 和 macOS
func cachePathFor(name string) string {
	// 优先使用 XDG_DATA_HOME 环境变量（Linux/macOS 标准）
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome != "" {
		return filepath.Join(xdgDataHome, appName, name+".json")
	}

	// 返回主数据目录的路径
	// 对于 Windows 系统，路径应为 `%LOCALAPPDATA%/crush/`
	// 对于 Linux 和 macOS 系统，路径应为 `$HOME/.local/share/crush/`
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(localAppData, appName, name+".json")
	}

	// 默认使用 ~/.local/share/appName/ 目录
	return filepath.Join(home.Dir(), ".local", "share", appName, name+".json")
}

// UpdateProviders 从指定源更新 Catwalk 提供者列表
// 参数 pathOrURL 可以是：
//   - "embedded": 使用嵌入的提供者
//   - HTTP/HTTPS URL: 从远程服务器获取
//   - 文件路径: 从本地文件读取
//   - 空字符串: 使用默认 URL 或环境变量 CATWALK_URL
func UpdateProviders(pathOrURL string) error {
	var providers []catwalk.Provider
	// 优先使用传入的路径，其次是环境变量，最后是默认 URL
	pathOrURL = cmp.Or(pathOrURL, os.Getenv("CATWALK_URL"), defaultCatwalkURL)

	switch {
	case pathOrURL == "embedded":
		// 使用嵌入的提供者列表
		providers = embedded.GetAll()
	case strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://"):
		// 从 HTTP/HTTPS URL 获取提供者
		var err error
		providers, err = catwalk.NewWithURL(pathOrURL).GetProviders(context.Background(), "")
		if err != nil {
			return fmt.Errorf("从 Catwalk 获取提供者失败: %w", err)
		}
	default:
		// 从本地文件读取提供者
		content, err := os.ReadFile(pathOrURL)
		if err != nil {
			return fmt.Errorf("读取文件失败: %w", err)
		}
		if err := json.Unmarshal(content, &providers); err != nil {
			return fmt.Errorf("反序列化提供者数据失败: %w", err)
		}
		if len(providers) == 0 {
			return fmt.Errorf("在提供的源中未找到提供者")
		}
	}

	// 将提供者列表保存到缓存
	if err := newCache[[]catwalk.Provider](cachePathFor("providers")).Store(providers); err != nil {
		return fmt.Errorf("保存提供者到缓存失败: %w", err)
	}

	slog.Info("提供者更新成功", "count", len(providers), "from", pathOrURL, "to", cachePathFor)
	return nil
}

// UpdateHyper 从指定 URL 更新 Hyper 提供者信息
// 参数 pathOrURL 可以是：
//   - "embedded": 使用嵌入的 Hyper 提供者
//   - HTTP/HTTPS URL: 从远程服务器获取
//   - 文件路径: 从本地文件读取
//   - 空字符串: 使用 Hyper.BaseURL()
func UpdateHyper(pathOrURL string) error {
	// 检查 Hyper 功能是否启用
	if !hyper.Enabled() {
		return fmt.Errorf("Hyper 未启用")
	}
	var provider catwalk.Provider
	// 优先使用传入的路径，其次是 Hyper 的基础 URL
	pathOrURL = cmp.Or(pathOrURL, hyper.BaseURL())

	switch {
	case pathOrURL == "embedded":
		// 使用嵌入的 Hyper 提供者
		provider = hyper.Embedded()
	case strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://"):
		// 从 HTTP/HTTPS URL 获取 Hyper 提供者
		client := realHyperClient{baseURL: pathOrURL}
		var err error
		provider, err = client.Get(context.Background(), "")
		if err != nil {
			return fmt.Errorf("从 Hyper 获取提供者失败: %w", err)
		}
	default:
		// 从本地文件读取 Hyper 提供者
		content, err := os.ReadFile(pathOrURL)
		if err != nil {
			return fmt.Errorf("读取文件失败: %w", err)
		}
		if err := json.Unmarshal(content, &provider); err != nil {
			return fmt.Errorf("反序列化提供者数据失败: %w", err)
		}
	}

	// 将 Hyper 提供者保存到缓存
	if err := newCache[catwalk.Provider](cachePathFor("hyper")).Store(provider); err != nil {
		return fmt.Errorf("保存 Hyper 提供者到缓存失败: %w", err)
	}

	slog.Info("Hyper 提供者更新成功", "from", pathOrURL, "to", cachePathFor("hyper"))
	return nil
}

// 全局同步器实例
var (
	catwalkSyncer = &catwalkSync{}  // Catwalk 提供者同步器
	hyperSyncer   = &hyperSync{}    // Hyper 提供者同步器
)

// Providers 返回提供者列表，考虑缓存结果以及是否启用自动更新
//
// 该函数将执行以下操作：
// 1. 如果禁用自动更新，则返回发布时嵌入的提供者
// 2. 加载缓存的提供者
// 3. 尝试获取最新的提供者列表，并返回新列表、缓存列表或嵌入列表（如果其他都失败）
//
// 该函数使用 sync.Once 确保提供者列表只加载一次，后续调用直接返回缓存结果
func Providers(cfg *Config) ([]catwalk.Provider, error) {
	providerOnce.Do(func() {
		var wg sync.WaitGroup
		var errs []error
		providers := csync.NewSlice[catwalk.Provider]()  // 使用并发安全的切片收集提供者
		autoupdate := !cfg.Options.DisableProviderAutoUpdate  // 自动更新标志

		// 设置 45 秒超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		// 并发获取 Catwalk 提供者
		wg.Go(func() {
			catwalkURL := cmp.Or(os.Getenv("CATWALK_URL"), defaultCatwalkURL)
			client := catwalk.NewWithURL(catwalkURL)
			path := cachePathFor("providers")
			catwalkSyncer.Init(client, path, autoupdate)

			items, err := catwalkSyncer.Get(ctx)
			if err != nil {
				catwalkURL := fmt.Sprintf("%s/v2/providers", cmp.Or(os.Getenv("CATWALK_URL"), defaultCatwalkURL))
				errs = append(errs, fmt.Errorf("Crush 无法从 %s 获取更新的提供者列表。考虑设置 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 以使用此 Crush 版本发布时捆绑的嵌入提供者。您也可以手动更新提供者。更多信息请参见 crush update-providers --help。\n\n原因: %w", catwalkURL, providerErr)) //nolint:staticcheck
				return
			}
			providers.Append(items...)
		})

		// 并发获取 Hyper 提供者（如果启用）
		wg.Go(func() {
			if !hyper.Enabled() {
				return
			}
			path := cachePathFor("hyper")
			hyperSyncer.Init(realHyperClient{baseURL: hyper.BaseURL()}, path, autoupdate)

			item, err := hyperSyncer.Get(ctx)
			if err != nil {
				errs = append(errs, fmt.Errorf("Crush 无法从 Hyper 获取更新的信息: %w", err)) //nolint:staticcheck
				return
			}
			providers.Append(item)
		})

		// 等待所有并发操作完成
		wg.Wait()

		// 收集所有提供者并合并错误
		providerList = slices.Collect(providers.Seq())
		providerErr = errors.Join(errs...)
	})
	return providerList, providerErr
}

// cache 是一个泛型缓存结构，用于存储和读取类型为 T 的数据
type cache[T any] struct {
	path string  // 缓存文件的路径
}

// newCache 创建一个新的缓存实例
func newCache[T any](path string) cache[T] {
	return cache[T]{path: path}
}

// Get 从缓存文件中读取数据
// 返回值：数据、ETag（用于缓存验证）、错误
func (c cache[T]) Get() (T, string, error) {
	var v T
	// 读取缓存文件
	data, err := os.ReadFile(c.path)
	if err != nil {
		return v, "", fmt.Errorf("读取提供者缓存文件失败: %w", err)
	}

	// 反序列化 JSON 数据
	if err := json.Unmarshal(data, &v); err != nil {
		return v, "", fmt.Errorf("从缓存反序列化提供者数据失败: %w", err)
	}

	// 返回数据和 ETag
	return v, etag.Of(data), nil
}

// Store 将数据保存到缓存文件
func (c cache[T]) Store(v T) error {
	slog.Info("将提供者数据保存到磁盘", "path", c.path)
	// 创建缓存目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("创建提供者缓存目录失败: %w", err)
	}

	// 序列化数据为 JSON
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("序列化提供者数据失败: %w", err)
	}

	// 写入文件（权限 0o644：所有者可读写，其他用户只读）
	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("写入提供者数据到缓存失败: %w", err)
	}
	return nil
}
