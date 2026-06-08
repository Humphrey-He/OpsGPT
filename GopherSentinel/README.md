# GopherSentinel

AI-powered DevOps Assistant - 基于 RAG 的智能运维助手

## 🚀 功能特性

- **RAG 知识库**: 自动解析和索引项目文档，支持 Markdown/PDF/TXT
- **智能问答**: 基于向量检索的精准问答，支持流式输出
- **多工具集成**: Prometheus 监控、Loki 日志、K8s 操作等
- **多 Agent 协作**: Master-Worker 架构处理复杂故障诊断
- **记忆系统**: 短期对话记忆 + 长期知识积累

## 📋 前置要求

- Go 1.21+
- Ollama (本地 LLM)
- Qdrant (向量数据库)
- Redis (可选，缓存用)

## 🛠️ 快速开始

### 1. 安装依赖

```bash
# 克隆项目
git clone <repo-url>
cd GopherSentinel

# 下载 Go 依赖
go mod download
```

### 2. 启动服务

```bash
# 启动 Ollama (确保模型已下载)
ollama serve
ollama pull llama3
ollama pull nomic-embed-text

# 启动 Qdrant
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### 3. 运行

```bash
# 构建
go build -o GopherSentinel ./cmd/cli

# 索引文档
./GopherSentinel ingest ./docs

# 提问
./GopherSentinel ask "这个项目的架构是什么？"

# 交互模式
./GopherSentinel ask -i
```

## 📁 项目结构

```
GopherSentinel/
├── cmd/cli/           # CLI 入口
├── internal/
│   ├── rag/          # RAG 链路
│   ├── memory/       # 记忆系统
│   └── agent/        # Agent 系统
├── pkg/
│   ├── parser/       # 文档解析
│   ├── vector/       # 向量存储
│   ├── llm/          # LLM 客户端
│   └── tools/        # 工具集
├── configs/          # 配置文件
└── docs/             # 文档目录
```

## ⚙️ 配置

编辑 `configs/config.yaml`:

```yaml
ollama:
  base_url: "http://localhost:11434"
  model: "llama3"
  embedding_model: "nomic-embed-text"

qdrant:
  url: "http://localhost:6333"
  collection: "gopher_sentinel"
  vector_size: 768

rag:
  chunk_size: 512
  chunk_overlap: 50
  top_k: 5
```

## 📖 开发计划

详见 [docs/02_详细开发计划.md](docs/02_详细开发计划.md)

- Week 1-2: 基础 RAG 知识库 (当前阶段)
- Week 3-4: 工具化与实时数据接入
- Week 5-6: 多智能体协作与记忆系统
- Week 7-8: 工程化打磨与评测

## 🧪 测试

```bash
go test ./... -v
```

## 📝 License

MIT
