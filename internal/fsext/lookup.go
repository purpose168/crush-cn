package fsext

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/purpose168/crush-cn/internal/home"
)

// Lookup 从指定目录开始向上搜索目标文件或目录，直到到达文件系统根目录。
// 该函数还会检查文件的所有权，以确保搜索不会跨越所有权边界。
// 对于所有权不匹配的情况，会跳过而不报错。
// 返回找到的目标的完整路径。
// 搜索范围包括起始目录本身。
func Lookup(dir string, targets ...string) ([]string, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	var found []string

	err := traverseUp(dir, func(cwd string, owner int) error {
		for _, target := range targets {
			fpath := filepath.Join(cwd, target)
			err := probeEnt(fpath, owner)

			// 权限被拒绝时跳过到下一个文件
			if errors.Is(err, os.ErrNotExist) ||
				errors.Is(err, os.ErrPermission) {
				continue
			}

			if err != nil {
				return fmt.Errorf("探测文件 %s 时出错: %w", fpath, err)
			}

			found = append(found, fpath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return found, nil
}

// LookupClosest 从指定目录开始向上搜索目标文件或目录，直到找到目标或到达根目录或主目录为止。
// 该函数还会检查文件的所有权，以确保搜索不会跨越所有权边界。
// 如果找到目标，返回目标的完整路径和 true；否则返回空字符串和 false。
// 搜索范围包括起始目录本身。
func LookupClosest(dir, target string) (string, bool) {
	var found string

	err := traverseUp(dir, func(cwd string, owner int) error {
		fpath := filepath.Join(cwd, target)

		err := probeEnt(fpath, owner)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("探测文件 %s 时出错: %w", fpath, err)
		}

		if cwd == home.Dir() {
			return filepath.SkipAll
		}

		found = fpath
		return filepath.SkipAll
	})

	return found, err == nil && found != ""
}

// traverseUp 从给定目录向上遍历，直到到达文件系统根目录。
// 它将当前目录的绝对路径和起始目录的所有者 ID 传递给回调函数。
// 所有者检查由用户自行处理。
func traverseUp(dir string, walkFn func(dir string, owner int) error) error {
	cwd, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("无法将当前工作目录转换为绝对路径: %w", err)
	}

	owner, err := Owner(dir)
	if err != nil {
		return fmt.Errorf("无法获取所有权: %w", err)
	}

	for {
		err := walkFn(cwd, owner)
		if err == nil || errors.Is(err, filepath.SkipDir) {
			parent := filepath.Dir(cwd)
			if parent == cwd {
				return nil
			}

			cwd = parent
			continue
		}

		if errors.Is(err, filepath.SkipAll) {
			return nil
		}

		return err
	}
}

// probeEnt 检查给定路径的实体是否存在且属于给定所有者
func probeEnt(fspath string, owner int) error {
	_, err := os.Stat(fspath)
	if err != nil {
		return fmt.Errorf("无法获取 %s 的文件状态: %w", fspath, err)
	}

	// 所有权检查绕过的特殊情况
	if owner == -1 {
		return nil
	}

	fowner, err := Owner(fspath)
	if err != nil {
		return fmt.Errorf("无法获取 %s 的所有权: %w", fspath, err)
	}

	if fowner != owner {
		return os.ErrPermission
	}

	return nil
}
