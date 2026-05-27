// Package domain 定义 k8s-ai-ops 平台的领域模型。
// 包含实体、值对象、仓储接口和领域服务。
// 领域层零外部依赖，不依赖任何框架或基础设施。
package domain

import "errors"

// 领域错误哨兵值。
var (
	ErrNotFound     = errors.New("not found")
	ErrDuplicate    = errors.New("duplicate")
	ErrInvalidInput = errors.New("invalid input")
	ErrDisabled     = errors.New("disabled")
	ErrForbidden    = errors.New("forbidden")
)
