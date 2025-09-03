# SPX Backend Documentation

## 项目概述

SPX Backend 是一个 Go 语言开发的后端服务，为 SPX 平台提供各种功能支持，包括项目管理、用户系统、AI 图片生成、文件存储等。

## 主要功能

### AI 图片生成系统
- **SVG 图片生成**：支持多种 AI 提供商（SVGIO、Recraft、OpenAI）
- **主题风格支持**：提供 9 种预定义主题，确保图片风格一致性
- **自动存储**：生成的图片自动存储到 Kodo 云存储

### 用户与项目管理
- **用户认证**：基于 Casdoor 的统一身份认证
- **项目管理**：支持项目创建、更新、发布版本管理
- **权限控制**：细粒度的权限管理和配额控制

### 文件存储服务
- **Kodo 集成**：七牛云对象存储服务集成
- **文件上传**：支持直接上传和代理上传
- **URL 管理**：内部 URL 与公网 URL 的转换

### AI 交互服务
- **AI Copilot**：代码生成和问答助手
- **工作流引擎**：支持复杂的 AI 处理流程
- **多轮对话**：支持上下文感知的对话系统

## 新增功能文档

### 🎨 SVG 主题功能
详细文档：[SVG_THEME_FEATURE.md](./SVG_THEME_FEATURE.md)

为 SVG 生成功能添加了主题支持：
- 9 种预定义风格主题（卡通、写实、极简等）
- 自动提示词增强，确保 AI 严格遵循风格要求
- 主题查询 API，前端可动态获取主题信息
- 中文支持，提供友好的用户界面

**核心接口**：
- `GET /themes` - 查询所有可用主题
- `POST /image/svg` - 生成 SVG（支持主题参数）
- `POST /image` - 生成图片元数据（支持主题参数）

### 💾 Kodo 存储集成
详细文档：[KODO_STORAGE_INTEGRATION.md](./KODO_STORAGE_INTEGRATION.md)

为 AI 图片生成功能集成了 Kodo 云存储：
- 自动存储生成的 SVG 和 PNG 图片
- 智能文件命名和去重机制
- 容错设计，存储失败不影响图片生成
- 支持多区域存储配置

**存储规范**：
- 路径格式：`ai-generated/{hash前8位}-{图片ID}.{扩展名}`
- URL 格式：`kodo://bucket/ai-generated/...`

## 技术架构

### 主要技术栈
- **语言**：Go 1.22+
- **Web 框架**：YAP (GoPlus Web Framework)
- **数据库**：MySQL + GORM
- **缓存**：Redis
- **存储**：七牛云 Kodo
- **认证**：Casdoor
- **AI 服务**：OpenAI API、Recraft API、SVG.IO API

### 项目结构
```
spx-backend/
├── cmd/spx-backend/          # 应用入口和 API 端点
├── internal/
│   ├── controller/           # 业务逻辑控制器
│   ├── model/               # 数据模型
│   ├── config/              # 配置管理
│   ├── svggen/              # SVG 生成服务
│   ├── copilot/             # AI Copilot 功能
│   ├── workflow/            # 工作流引擎
│   └── docs/                # 内部文档
├── docs/                    # 项目文档
└── README.md
```

## 配置要求

### 环境变量
```bash
# 数据库配置
GOP_SPX_DSN=mysql_connection_string

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_password

# Kodo 存储配置
KODO_AK=your_access_key
KODO_SK=your_secret_key
KODO_BUCKET=your_bucket_name
KODO_BUCKET_REGION=your_region
KODO_BASE_URL=https://your_domain.com

# AI 服务配置
OPENAI_API_KEY=your_openai_key
OPENAI_API_ENDPOINT=https://api.openai.com/v1

# Casdoor 认证配置
GOP_CASDOOR_ENDPOINT=https://your_casdoor_endpoint
```

## 快速开始

### 1. 环境准备
```bash
# 安装 Go 1.22+
go version

# 克隆项目
git clone <repository_url>
cd spx-backend
```

### 2. 配置环境变量
```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置文件
vim .env
```

### 3. 运行项目
```bash
# 安装依赖
go mod download

# 构建项目
go build ./cmd/spx-backend

# 运行服务
./spx-backend
```

### 4. API 测试
```bash
# 健康检查
curl http://localhost:8080/health

# 获取主题列表
curl http://localhost:8080/themes

# 生成 SVG 图片
curl -X POST http://localhost:8080/image/svg \
  -H "Content-Type: application/json" \
  -d '{"prompt":"一只可爱的小猫","theme":"cartoon","provider":"svgio"}'
```

## 开发指南

### 添加新功能
1. 在 `internal/controller/` 中添加业务逻辑
2. 在 `cmd/spx-backend/` 中添加 API 端点文件
3. 更新相关配置和文档
4. 编写单元测试

### 测试
```bash
# 运行所有测试
go test ./...

# 运行指定模块测试
go test ./internal/controller

# 运行测试并查看覆盖率
go test -cover ./...
```

### 代码规范
- 使用 `gofmt` 格式化代码
- 遵循 Go 官方代码规范
- 为公共函数编写注释
- 编写单元测试覆盖核心逻辑

## API 文档

### 核心接口

#### 图片生成
- `POST /image/svg` - 生成 SVG 图片
- `POST /image` - 生成图片元数据
- `GET /themes` - 获取主题列表

#### 文件管理
- `GET /util/upinfo` - 获取文件上传信息
- `POST /util/fileurls` - 生成文件访问 URL

#### 用户认证
- `GET /user` - 获取当前用户信息
- `GET /user/{username}` - 获取指定用户信息

#### 项目管理
- `GET /projects/list` - 获取项目列表
- `POST /project` - 创建项目
- `PUT /project/{owner}/{name}` - 更新项目

详细的 API 参数和响应格式请参考各个端点的 `.yap` 文件。

## 部署指南

### Docker 部署
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o spx-backend ./cmd/spx-backend

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/spx-backend .
CMD ["./spx-backend"]
```

### 生产环境配置
- 配置反向代理（Nginx）
- 设置 HTTPS 证书
- 配置日志轮转
- 设置监控和告警

## 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查数据库服务状态
   - 验证连接字符串格式
   - 确认网络连通性

2. **图片生成失败**
   - 检查 AI 服务 API 密钥
   - 验证网络连接
   - 查看错误日志

3. **文件上传失败**
   - 检查 Kodo 配置参数
   - 验证存储桶权限
   - 确认文件大小限制

### 日志分析
```bash
# 查看应用日志
tail -f /var/log/spx-backend.log

# 过滤错误日志
grep "ERROR" /var/log/spx-backend.log

# 分析特定功能日志
grep "SVG generation" /var/log/spx-backend.log
```

## 贡献指南

### 提交规范
- 使用语义化的提交信息
- 提交前运行测试确保通过
- 更新相关文档

### Issue 和 PR
- 详细描述问题或功能需求
- 提供复现步骤或使用示例
- 遵循代码审查流程

## 版本历史

### v2.x.x (当前版本)
- ✨ 新增 SVG 主题功能支持
- ✨ 集成 Kodo 云存储自动备份
- 🐛 修复图片生成的稳定性问题
- 📚 完善项目文档

### v1.x.x
- 🎉 初始版本发布
- ✨ 基础项目管理功能
- ✨ 用户认证系统
- ✨ AI Copilot 集成

## 许可证

本项目采用 MIT 许可证，详细信息请参考 LICENSE 文件。

---

更多技术细节请参考各个功能模块的专门文档。如有问题，请通过 Issue 提交反馈。