package tools

import (
	"cmp"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/filepathext"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/permission"
)

type LSParams struct {
	Path   string   `json:"path,omitempty" description:"要列出的目录路径（默认为当前工作目录）"`
	Ignore []string `json:"ignore,omitempty" description:"要忽略的 glob 模式列表"`
	Depth  int      `json:"depth,omitempty" description:"要遍历的最大深度"`
}

type LSPermissionsParams struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore"`
	Depth  int      `json:"depth"`
}

type NodeType string

const (
	NodeTypeFile      NodeType = "file"
	NodeTypeDirectory NodeType = "directory"
)

type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     NodeType    `json:"type"`
	Children []*TreeNode `json:"children,omitempty"`
}

type LSResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

const (
	LSToolName = "ls"
	maxLSFiles = 1000
)

//go:embed ls.md
var lsDescription []byte

func NewLsTool(permissions permission.Service, workingDir string, lsConfig config.ToolLs) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		LSToolName,
		string(lsDescription),
		func(ctx context.Context, params LSParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			searchPath, err := fsext.Expand(cmp.Or(params.Path, workingDir))
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("扩展路径错误: %v", err)), nil
			}

			searchPath = filepathext.SmartJoin(workingDir, searchPath)

			// 检查目录是否在工作目录外，如需请求权限
			absWorkingDir, err := filepath.Abs(workingDir)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("解析工作目录错误: %v", err)), nil
			}

			absSearchPath, err := filepath.Abs(searchPath)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("解析搜索路径错误: %v", err)), nil
			}

			relPath, err := filepath.Rel(absWorkingDir, absSearchPath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				// 目录在工作目录外，请求权限
				sessionID := GetSessionFromContext(ctx)
				if sessionID == "" {
					return fantasy.ToolResponse{}, fmt.Errorf("访问工作目录外的目录需要会话 ID")
				}

				granted, err := permissions.Request(ctx,
					permission.CreatePermissionRequest{
						SessionID:   sessionID,
						Path:        absSearchPath,
						ToolCallID:  call.ID,
						ToolName:    LSToolName,
						Action:      "list",
						Description: fmt.Sprintf("列出工作目录外的目录: %s", absSearchPath),
						Params:      LSPermissionsParams(params),
					},
				)
				if err != nil {
					return fantasy.ToolResponse{}, err
				}
				if !granted {
					return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
				}
			}

			output, metadata, err := ListDirectoryTree(searchPath, params, lsConfig)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output),
				metadata,
			), nil
		})
}

func ListDirectoryTree(searchPath string, params LSParams, lsConfig config.ToolLs) (string, LSResponseMetadata, error) {
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return "", LSResponseMetadata{}, fmt.Errorf("路径不存在: %s", searchPath)
	}

	depth, limit := lsConfig.Limits()
	maxFiles := cmp.Or(limit, maxLSFiles)
	files, truncated, err := fsext.ListDirectory(
		searchPath,
		params.Ignore,
		cmp.Or(params.Depth, depth),
		maxFiles,
	)
	if err != nil {
		return "", LSResponseMetadata{}, fmt.Errorf("列出目录错误: %w", err)
	}

	metadata := LSResponseMetadata{
		NumberOfFiles: len(files),
		Truncated:     truncated,
	}
	tree := createFileTree(files, searchPath)

	var output string
	if truncated {
		output = fmt.Sprintf("目录中有超过 %d 个文件。使用更具体的路径或使用 Glob 工具查找特定文件。以下包含前 %[1]d 个文件和目录。\n", maxFiles)
	}
	if depth > 0 {
		output = fmt.Sprintf("目录树显示深度最多为 %d。使用更高的深度和特定路径查看更多级别。\n", cmp.Or(params.Depth, depth))
	}
	return output + "\n" + printTree(tree, searchPath), metadata, nil
}

func createFileTree(sortedPaths []string, rootPath string) []*TreeNode {
	root := []*TreeNode{}
	pathMap := make(map[string]*TreeNode)

	for _, path := range sortedPaths {
		relativePath := strings.TrimPrefix(path, rootPath)
		parts := strings.Split(relativePath, string(filepath.Separator))
		currentPath := ""
		var parentPath string

		var cleanParts []string
		for _, part := range parts {
			if part != "" {
				cleanParts = append(cleanParts, part)
			}
		}
		parts = cleanParts

		if len(parts) == 0 {
			continue
		}

		for i, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			if _, exists := pathMap[currentPath]; exists {
				parentPath = currentPath
				continue
			}

			isLastPart := i == len(parts)-1
			isDir := !isLastPart || strings.HasSuffix(relativePath, string(filepath.Separator))
			nodeType := NodeTypeFile
			if isDir {
				nodeType = NodeTypeDirectory
			}
			newNode := &TreeNode{
				Name:     part,
				Path:     currentPath,
				Type:     nodeType,
				Children: []*TreeNode{},
			}

			pathMap[currentPath] = newNode

			if i > 0 && parentPath != "" {
				if parent, ok := pathMap[parentPath]; ok {
					parent.Children = append(parent.Children, newNode)
				}
			} else {
				root = append(root, newNode)
			}

			parentPath = currentPath
		}
	}

	return root
}

func printTree(tree []*TreeNode, rootPath string) string {
	var result strings.Builder

	result.WriteString("- ")
	result.WriteString(filepath.ToSlash(rootPath))
	if rootPath[len(rootPath)-1] != '/' {
		result.WriteByte('/')
	}
	result.WriteByte('\n')

	for _, node := range tree {
		printNode(&result, node, 1)
	}

	return result.String()
}

func printNode(builder *strings.Builder, node *TreeNode, level int) {
	indent := strings.Repeat("  ", level)

	nodeName := node.Name
	if node.Type == NodeTypeDirectory {
		nodeName = nodeName + "/"
	}

	fmt.Fprintf(builder, "%s- %s\n", indent, nodeName)

	if node.Type == NodeTypeDirectory && len(node.Children) > 0 {
		for _, child := range node.Children {
			printNode(builder, child, level+1)
		}
	}
}
