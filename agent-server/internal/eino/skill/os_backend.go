package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk/filesystem"
)

// OSBackend 是 filesystem.Backend 的真实操作系统实现。
type OSBackend struct {
	mu sync.RWMutex
}

// NewOSBackend 创建基于真实文件系统的 Backend。
func NewOSBackend() *OSBackend {
	return &OSBackend{}
}

// LsInfo 列出指定路径下的文件和目录信息。
func (b *OSBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", req.Path, err)
	}

	var result []filesystem.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, filesystem.FileInfo{
			Path:       filepath.Join(req.Path, entry.Name()),
			IsDir:      entry.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().Format(time.RFC3339Nano),
		})
	}
	return result, nil
}

// Read 读取文件内容，支持 offset/limit 按行截取。
func (b *OSBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", req.FilePath, err)
	}

	content := string(data)
	offset := req.Offset - 1
	if offset < 0 {
		offset = 0
	}
	limit := req.Limit

	if offset == 0 && limit <= 0 {
		return &filesystem.FileContent{Content: content}, nil
	}

	lines := strings.Split(content, "\n")
	if offset >= len(lines) {
		return &filesystem.FileContent{}, nil
	}

	if limit <= 0 || offset+limit > len(lines) {
		return &filesystem.FileContent{Content: strings.Join(lines[offset:], "\n")}, nil
	}

	return &filesystem.FileContent{Content: strings.Join(lines[offset:offset+limit], "\n")}, nil
}

// GrepRaw 在文件中按正则搜索匹配行。
func (b *OSBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	if req.Pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	re, err := regexp.Compile(req.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var matches []filesystem.GrepMatch

	err = filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if req.Glob != "" {
			matched, _ := filepath.Match(req.Glob, filepath.Base(path))
			if !matched {
				return nil
			}
		}
		if req.FileType != "" {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if ext != req.FileType {
				return nil
			}
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, filesystem.GrepMatch{
					Path:    path,
					Line:    i + 1,
					Content: line,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("grep walk %s: %w", req.Path, err)
	}

	return matches, nil
}

// GlobInfo 按 glob 模式匹配文件并返回文件信息。
func (b *OSBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	pattern := filepath.Join(req.Path, req.Pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}

	var result []filesystem.FileInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		result = append(result, filesystem.FileInfo{
			Path:       match,
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().Format(time.RFC3339Nano),
		})
	}
	return result, nil
}

// Write 创建或覆盖文件内容。
func (b *OSBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	dir := filepath.Dir(req.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	if err := os.WriteFile(req.FilePath, []byte(req.Content), 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", req.FilePath, err)
	}
	return nil
}

// Edit 替换文件中的字符串出现。
func (b *OSBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", req.FilePath, err)
	}

	content := string(data)
	if !strings.Contains(content, req.OldString) {
		return fmt.Errorf("oldString not found in file: %s", req.FilePath)
	}

	if !req.ReplaceAll {
		firstIndex := strings.Index(content, req.OldString)
		if firstIndex != -1 {
			if strings.Contains(content[firstIndex+len(req.OldString):], req.OldString) {
				return fmt.Errorf("multiple occurrences of oldString found, but ReplaceAll is false")
			}
		}
	}

	var newContent string
	if req.ReplaceAll {
		newContent = strings.ReplaceAll(content, req.OldString, req.NewString)
	} else {
		newContent = strings.Replace(content, req.OldString, req.NewString, 1)
	}

	if err := os.WriteFile(req.FilePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", req.FilePath, err)
	}
	return nil
}
