package handler

import "github.com/sirupsen/logrus"

// log 是 handler 包的共享日志实例，所有 handler 函数共用。
// 使用 component 字段标识来源为 mcp-server。
var log = logrus.WithField("component", "mcp-server")
