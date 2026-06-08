# 🔭 GopherSentinel

> 企业级 AI 智能运维与研发助手 | Enterprise AI Operations & Development Assistant

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/Humphrey-He/OpsGPT/ci.yml)](https://github.com/Humphrey-He/OpsGPT/actions)

## ✨ 特性

- 🤖 **RAG 智能问答** - 基于混合检索和 Rerank 的高精度知识库问答
- 🔧 **Multi-Agent 协作** - 多智能体并行分析日志、指标、文档
- 📡 **Function Calling** - 自然语言驱动 Prometheus、Loki、K8s 等系统
- 🧠 **分层记忆系统** - 工作记忆 → 短期记忆 (Redis) → 长期记忆 (向量库)
- 🔒 **企业级安全** - 限流、审计、危险操作确认
- 📊 **全链路可观测** - Langfuse 追踪思维链，Zap 结构化日志

## 📖 文档

- [项目规划文档](./docs/)
- [快速开始指南](./docs/05_快速开始指南.md)
- [技术架构设计](./docs/03_技术架构设计.md)
- [面试题问答手册](./docs/07_面试题问答手册.md)

## 🚀 快速开始

### 使用 Docker Compose

```bash
# 克隆项目
git clone https://github.com/Humphrey-He/OpsGPT.git
cd OpsGPT/GopherSentinel

# 启动所有服务
docker-compose up -d

# 运行 CLI
docker-compose exec app ./GopherSentinel ask "帮我分析订单服务"
```

### 手动部署

```bash
# 1. 安装依赖
go mod download

# 2. 配置环境
cp .env.example .env
# 编辑 .env 配置

# 3. 启动服务 (Qdrant, Redis, Ollama)
docker-compose up -d qdrant redis ollama

# 4. 构建并运行
make build
./bin/GopherSentinel ask "这个项目怎么部署？"
```

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────────┐
│                    GopherSentinel                       │
├─────────────────────────────────────────────────────────┤
│  CLI / HTTP API                                        │
├─────────────────────────────────────────────────────────┤
│  Master Agent (意图识别 → 任务调度)                    │
├─────────────────────────────────────────────────────────┤
│  Log-Agent │ Metric-Agent │ Doc-Agent │ Custom-Agent   │
├─────────────────────────────────────────────────────────┤
│  Tools: Prometheus | Loki | K8s | HTTP | DB           │
├─────────────────────────────────────────────────────────┤
│  RAG: Embedding → Hybrid Search → Rerank → Generate   │
├─────────────────────────────────────────────────────────┤
│  Memory: Working → Short-term (Redis) → Long-term     │
└─────────────────────────────────────────────────────────┘
```

## 📈 技术栈

| 层级 | 技术选型 | 说明 |
|-----|---------|------|
| **LLM** | Ollama / OpenAI | 本地或云端推理 |
| **向量库** | Qdrant | 高性能向量检索 |
| **缓存** | Redis | 短期记忆、限流 |
| **追踪** | Langfuse | Agent 思维链追踪 |
| **日志** | Zap | 结构化日志 |
| **CLI** | Cobra | 命令行框架 |

## 📂 项目结构

```
GopherSentinel/
├── cmd/cli/                    # CLI 入口
├── internal/
│   ├── agent/                  # Agent 核心 (Master, ReAct, SubAgents)
│   ├── rag/                    # RAG 链路 (检索, Rerank)
│   └── memory/                 # 记忆系统 (Working, Short, Long)
├── pkg/
│   ├── llm/                    # LLM 客户端 (Ollama, OpenAI)
│   ├── tools/                  # 工具系统 (Prometheus, Loki, K8s)
│   ├── parser/                 # 文档解析 (Markdown, PDF, Text)
│   ├── vector/                 # 向量存储 (Qdrant)
│   ├── observability/           # 可观测性 (Langfuse)
│   └── security/               # 安全模块 (限流, 注入检测)
├── configs/                    # 配置文件
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 🧪 测试

```bash
# 运行所有测试
make test

# 带覆盖率
make test-coverage

# E2E 测试
make e2e
```

## 📋 开发阶段

| 阶段 | 状态 | 说明 |
|-----|------|------|
| Phase 1 | ✅ 完成 | 基础 RAG + CLI |
| Phase 2 | ✅ 完成 | 工具集成 + Function Calling |
| Phase 3 | ✅ 完成 | Multi-Agent + 记忆系统 |
| Phase 4 | ✅ 完成 | 安全 + 可观测性 + 部署 |
| 文档 | ✅ 完成 | 完整项目文档 |

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License
