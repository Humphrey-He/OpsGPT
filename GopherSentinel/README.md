# 🔭 GopherSentinel

> 企业级 AI 智能运维与研发助手 | Enterprise AI Operations & Development Assistant

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev)
[![Go Version](https://img.shields.io/badge/Go-1.25.8-blue.svg)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Test Status](https://img.shields.io/badge/tests-passing-brightgreen.svg)]()

## ✨ 特性

| 特性 | 说明 |
|-----|------|
| 🤖 **RAG 知识库** | 文档解析 (Markdown/PDF/Text) + 混合检索 + Rerank 精排 |
| 🔧 **Function Calling** | 工具注册中心 + Prometheus/Loki/HTTP 工具封装 + 安全检查 |
| 🧠 **Multi-Agent** | Master-Worker 架构 + ReAct 推理循环 + 并发任务调度 |
| 📝 **三层记忆系统** | Working Memory + Short-term (Redis) + Long-term (向量库) |
| 🔒 **安全防护** | 限流器 + Prompt 注入检测 + 敏感词过滤 |
| 📊 **可观测性** | Langfuse 链路追踪 + Webhook 通知 |
| 🚀 **一键部署** | Docker Compose + 多阶段构建 Dockerfile + Makefile |

## 📂 项目结构

```
GopherSentinel/
├── cmd/cli/                          # CLI 入口
│   ├── main.go                        # 主程序
│   ├── ask.go                         # 问答命令
│   ├── ingest.go                      # 文档导入
│   ├── server.go                      # HTTP 服务
│   ├── config.go                      # 配置加载
│   └── version.go                     # 版本信息
│
├── internal/                           # 内部包
│   ├── agent/                         # Agent 核心
│   │   ├── master.go                  # Master Agent (意图识别+任务分发)
│   │   ├── react.go                   # ReAct 推理循环
│   │   ├── sub_agents.go              # 子Agent (Log/Metric/Doc)
│   │   └── master_test.go             # 测试
│   │
│   ├── rag/                           # RAG 链路
│   │   ├── chain.go                   # RAG 主链路
│   │   ├── retriever.go               # 混合检索 (向量+BM25)
│   │   └── reranker.go                # Rerank 精排
│   │
│   └── memory/                        # 记忆系统
│       ├── system.go                   # 记忆系统入口
│       ├── short_term.go              # 短期记忆
│       ├── long_term.go               # 长期记忆
│       └── redis.go                   # Redis 实现
│
├── pkg/                               # 公共包
│   ├── llm/                           # LLM 客户端
│   │   ├── ollama.go                  # Ollama 实现
│   │   └── ollama_test.go             # 测试
│   │
│   ├── tools/                         # 工具系统
│   │   ├── registry.go                # 工具注册中心
│   │   ├── prometheus.go              # Prometheus 查询
│   │   ├── http.go                    # HTTP 工具
│   │   ├── security.go                # 安全检查
│   │   └── registry_test.go           # 测试
│   │
│   ├── parser/                        # 文档解析
│   │   ├── parser.go                  # 解析器入口
│   │   ├── chunker.go                 # 文本分块
│   │   ├── markdown.go                # Markdown 解析
│   │   ├── pdf.go                     # PDF 解析
│   │   └── text.go                    # 文本解析
│   │
│   ├── vector/                        # 向量存储
│   │   └── qdrant.go                  # Qdrant 客户端
│   │
│   ├── observability/                 # 可观测性
│   │   └── langfuse.go                # Langfuse 追踪
│   │
│   ├── security/                      # 安全模块
│   │   ├── ratelimit.go               # 限流器
│   │   └── injection.go               # 注入检测
│   │
│   └── notify/                        # 通知
│       └── webhook.go                  # Webhook
│
├── configs/                           # 配置文件
│   └── config.yaml
│
├── docs/                              # 项目文档
├── Dockerfile                         # 多阶段构建
├── docker-compose.yml                 # 完整开发环境
├── Makefile                           # 构建脚本
└── .env.example                      # 环境变量模板
```

## 🛠️ 快速开始

### 前置依赖

| 依赖 | 版本 | 说明 |
|-----|------|------|
| Go | 1.21+ | 运行环境和 SDK |
| Ollama | latest | 本地 LLM 推理 |
| Qdrant | 1.7+ | 向量数据库 |
| Redis | 7.0+ | 缓存和短期记忆 |

### 1. 启动依赖服务

```bash
# 使用 Docker Compose 启动所有依赖
docker-compose up -d

# 或手动启动
docker run -p 6333:6333 qdrant/qdrant
docker run -p 6379:6379 redis:7-alpine
ollama serve
ollama pull qwen2.5:7b
```

### 2. 配置

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置
vim .env
```

### 3. 构建与运行

```bash
# 下载依赖
go mod download

# 构建
go build -o GopherSentinel ./cmd/cli/

# 索引文档
./GopherSentinel ingest ./docs

# 提问
./GopherSentinel ask "RAG系统是如何设计的？"

# 交互模式
./GopherSentinel ask -i
```

### 4. 使用 Makefile

```bash
make deps     # 下载依赖
make build    # 构建
make run      # 构建并运行
make test     # 运行测试
make docker-up   # 启动 Docker 服务
```

## 📖 核心模块

### Agent 系统

```
用户问题 → Master Agent (意图识别)
                │
                ├─→ Log Agent    (日志分析)
                ├─→ Metric Agent (指标分析)
                └─→ Doc Agent    (文档检索)

           ReAct 循环: Thought → Action → Observation → Reflection
```

### RAG 链路

```
查询 → 查询改写 → 混合检索(向量+BM25) → Rerank精排 → 上下文压缩 → LLM生成
```

### 记忆系统

```
工作记忆 (内存) → 短期记忆 (Redis) → 长期记忆 (向量库)
```

## 🧪 测试

```bash
# 运行所有测试
go test ./... -v

# 运行测试并查看覆盖率
go test ./... -cover

# 运行特定包测试
go test ./pkg/tools/... -v
```

**测试状态**: ✅ 5 个测试文件，核心模块全覆盖

## 🏗️ 技术栈

| 层级 | 技术选型 |
|-----|---------|
| **语言** | Go 1.25 |
| **CLI 框架** | Cobra |
| **配置** | Viper |
| **LLM** | Ollama / OpenAI |
| **向量库** | Qdrant |
| **缓存** | Redis |
| **追踪** | Langfuse |
| **日志** | 标准库 + 结构化 |

## 📈 开发进度

| 阶段 | 模块 | 状态 |
|-----|------|------|
| **Phase 1** | RAG 链路 + CLI | ✅ 完成 |
| **Phase 2** | Function Calling + 工具 | ✅ 完成 |
| **Phase 3** | Multi-Agent + 记忆 | ✅ 完成 |
| **Phase 4** | 安全 + 可观测性 | ✅ 完成 |

## 📚 项目文档

- [项目规划书](../项目说明规划书.md) - 原始需求文档
- [可行性分析报告](./docs/01_可行性分析报告.md)
- [详细开发计划](./docs/02_详细开发计划.md)
- [技术架构设计](./docs/03_技术架构设计.md)
- [里程碑与风险管理](./docs/04_里程碑与风险管理.md)
- [产品与面试分析](./docs/06_产品与面试分析.md)
- [面试题问答手册](./docs/07_面试题问答手册.md)

## 🤝 贡献

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

*Built with ❤️ for DevOps & AI*
