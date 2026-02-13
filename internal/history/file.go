package history

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/purpose168/crush-cn/internal/db"
	"github.com/purpose168/crush-cn/internal/pubsub"
)

// 初始版本号常量
const (
	InitialVersion = 0
)

// File 文件结构体，表示一个文件的历史版本记录
type File struct {
	ID        string // 文件唯一标识符
	SessionID string // 所属会话ID
	Path      string // 文件路径
	Content   string // 文件内容
	Version   int64  // 版本号
	CreatedAt int64  // 创建时间戳
	UpdatedAt int64  // 更新时间戳
}

// Service 文件服务接口，管理会话的文件版本和历史记录
type Service interface {
	pubsub.Subscriber[File]
	// Create 创建新文件记录
	Create(ctx context.Context, sessionID, path, content string) (File, error)

	// CreateVersion 创建文件的新版本，版本号自动递增
	CreateVersion(ctx context.Context, sessionID, path, content string) (File, error)

	// Get 根据ID获取文件记录
	Get(ctx context.Context, id string) (File, error)
	// GetByPathAndSession 根据路径和会话ID获取文件记录
	GetByPathAndSession(ctx context.Context, path, sessionID string) (File, error)
	// ListBySession 列出指定会话的所有文件记录
	ListBySession(ctx context.Context, sessionID string) ([]File, error)
	// ListLatestSessionFiles 列出指定会话的最新版本文件
	ListLatestSessionFiles(ctx context.Context, sessionID string) ([]File, error)
	// Delete 删除指定ID的文件记录
	Delete(ctx context.Context, id string) error
	// DeleteSessionFiles 删除指定会话的所有文件记录
	DeleteSessionFiles(ctx context.Context, sessionID string) error
}

// service 文件服务实现结构体
type service struct {
	*pubsub.Broker[File] // 发布订阅代理
	db *sql.DB           // 数据库连接
	q  *db.Queries       // 数据库查询对象
}

// NewService 创建新的文件服务实例
func NewService(q *db.Queries, db *sql.DB) Service {
	return &service{
		Broker: pubsub.NewBroker[File](),
		q:      q,
		db:     db,
	}
}

// Create 创建新文件记录，使用初始版本号
func (s *service) Create(ctx context.Context, sessionID, path, content string) (File, error) {
	return s.createWithVersion(ctx, sessionID, path, content, InitialVersion)
}

// CreateVersion 创建文件的新版本，版本号自动递增
// 如果该路径不存在之前的版本，则创建初始版本
// 提供的内容将作为新版本存储
func (s *service) CreateVersion(ctx context.Context, sessionID, path, content string) (File, error) {
	// 获取该路径的最新版本
	files, err := s.q.ListFilesByPath(ctx, path)
	if err != nil {
		return File{}, err
	}

	if len(files) == 0 {
		// 没有之前的版本，创建初始版本
		return s.Create(ctx, sessionID, path, content)
	}

	// 获取最新版本，文件按版本号降序、创建时间降序排列
	latestFile := files[0]
	nextVersion := latestFile.Version + 1

	return s.createWithVersion(ctx, sessionID, path, content, nextVersion)
}

// createWithVersion 使用指定版本号创建文件记录
func (s *service) createWithVersion(ctx context.Context, sessionID, path, content string, version int64) (File, error) {
	// 事务冲突的最大重试次数
	const maxRetries = 3
	var file File
	var err error

	// 事务冲突重试循环
	for attempt := range maxRetries {
		// 开启事务
		tx, txErr := s.db.BeginTx(ctx, nil)
		if txErr != nil {
			return File{}, fmt.Errorf("开启事务失败: %w", txErr)
		}

		// 使用事务创建新的查询实例
		qtx := s.q.WithTx(tx)

		// 尝试在事务中创建文件
		dbFile, txErr := qtx.CreateFile(ctx, db.CreateFileParams{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Path:      path,
			Content:   content,
			Version:   version,
		})
		if txErr != nil {
			// 回滚事务
			tx.Rollback()

			// 检查是否为唯一性约束冲突
			if strings.Contains(txErr.Error(), "UNIQUE constraint failed") {
				if attempt < maxRetries-1 {
					// 如果还有重试机会，递增版本号后重试
					version++
					continue
				}
			}
			return File{}, txErr
		}

		// 提交事务
		if txErr = tx.Commit(); txErr != nil {
			return File{}, fmt.Errorf("提交事务失败: %w", txErr)
		}

		file = s.fromDBItem(dbFile)
		s.Publish(pubsub.CreatedEvent, file)
		return file, nil
	}

	return file, err
}

// Get 根据ID获取文件记录
func (s *service) Get(ctx context.Context, id string) (File, error) {
	dbFile, err := s.q.GetFile(ctx, id)
	if err != nil {
		return File{}, err
	}
	return s.fromDBItem(dbFile), nil
}

// GetByPathAndSession 根据路径和会话ID获取文件记录
func (s *service) GetByPathAndSession(ctx context.Context, path, sessionID string) (File, error) {
	dbFile, err := s.q.GetFileByPathAndSession(ctx, db.GetFileByPathAndSessionParams{
		Path:      path,
		SessionID: sessionID,
	})
	if err != nil {
		return File{}, err
	}
	return s.fromDBItem(dbFile), nil
}

// ListBySession 列出指定会话的所有文件记录
func (s *service) ListBySession(ctx context.Context, sessionID string) ([]File, error) {
	dbFiles, err := s.q.ListFilesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromDBItem(dbFile)
	}
	return files, nil
}

// ListLatestSessionFiles 列出指定会话的最新版本文件
func (s *service) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]File, error) {
	dbFiles, err := s.q.ListLatestSessionFiles(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromDBItem(dbFile)
	}
	return files, nil
}

// Delete 删除指定ID的文件记录
func (s *service) Delete(ctx context.Context, id string) error {
	file, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	err = s.q.DeleteFile(ctx, id)
	if err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, file)
	return nil
}

// DeleteSessionFiles 删除指定会话的所有文件记录
func (s *service) DeleteSessionFiles(ctx context.Context, sessionID string) error {
	files, err := s.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, file := range files {
		err = s.Delete(ctx, file.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// fromDBItem 将数据库文件模型转换为业务文件模型
func (s *service) fromDBItem(item db.File) File {
	return File{
		ID:        item.ID,
		SessionID: item.SessionID,
		Path:      item.Path,
		Content:   item.Content,
		Version:   item.Version,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
