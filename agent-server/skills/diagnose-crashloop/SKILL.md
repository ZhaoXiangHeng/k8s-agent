---
name: diagnose-crashloop
description: 诊断 Pod CrashLoopBackOff 状态，分析容器退出原因、检查资源限制、查看事件日志。当用户报告 Pod 反复重启或处于 CrashLoopBackOff 状态时使用此技能。
---
# CrashLoopBackOff 诊断流程

当 Pod 处于 CrashLoopBackOff 状态时，按以下步骤诊断：

## 步骤
1. 使用 `get_pods` 获取目标 Pod 的完整 YAML，关注 `restartCount`、`lastState` 和容器退出码
2. 使用 `get_pod_logs` 查看容器日志（包括 `previous: true` 以获取上一次崩溃的日志）
3. 使用 `get_events` 查看该 Pod 的相关事件，重点关注 `OOMKilled`、`ImagePullBackOff`、`LivenessProbeFailed` 等事件
4. 检查资源限制（`resources.limits` 和 `resources.requests`），确认是否因 OOM 或 CPU 节流导致崩溃
5. 检查探针配置（`livenessProbe`、`readinessProbe`、`startupProbe`），确认探针超时和阈值是否合理

## 常见根因
- **OOMKilled**: 内存不足，需增加 memory limit 或优化应用内存使用
- **探针失败**: 健康检查配置不当或应用启动过慢，需调整 `initialDelaySeconds` 或 `failureThreshold`
- **镜像拉取失败**: 镜像不存在或仓库认证问题
- **配置错误**: 环境变量、挂载卷、ConfigMap/Secret 缺失或格式错误
- **应用内部错误**: 代码 bug、数据库连接失败、依赖服务不可达

## 输出要求
综合以上信息，给出：
1. 根因分析
2. 相关证据（日志片段、事件内容）
3. 修复建议
