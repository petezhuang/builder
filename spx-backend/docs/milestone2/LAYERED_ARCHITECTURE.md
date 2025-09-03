# SPX-Backend AI生图与推荐服务分层架构文档

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                        接入层 (API Layer)                     │
│  post_image.yap | post_image_svg.yap | post_image_recommend.yap │
│     /images          /images/svg         /images/recommend     │
├─────────────────────────────────────────────────────────────┤
│                       应用层 (Application Layer)              │
│     svg.go (AI生图控制器)    |    image_recommend.go (推荐控制器)   │
│ GenerateSVG() GenerateImage() | RecommendImages() generateAISVGs() │
│      业务流程编排 | 错误处理 | 服务组合 | 混合推荐策略              │
├─────────────────────────────────────────────────────────────┤
│                       领域层 (Domain Layer)                   │
│               ServiceManager (svggen/) - 服务管理器             │
│   svgio.go    |   recraft.go   |   openai.go   |  translate.go │
│  专业SVG生成   |  高质量矢量图   |  通用AI绘图   |    翻译服务      │
│              Provider策略 | 生成规则 | 推荐算法                │
├─────────────────────────────────────────────────────────────┤
│                     基础设施层 (Infrastructure Layer)          │
│ model/ (数据模型) | config/ (配置) | log/ (日志) | kodoClient (存储) │
│ AIResource | Label | KodoConfig | Logger | Credentials | Storage │
│        MySQL数据库 | 七牛云存储 | 配置管理 | 日志服务            │
└─────────────────────────────────────────────────────────────┘
                                ↕
        ┌─────────────────外部依赖服务─────────────────┐
        │ spx-algorithm(语义搜索) | OpenAI/Recraft/SVGIO │
        │     向量数据库        |      Kodo云存储        │
        └─────────────────────────────────────────────┘
```

## 分层详细设计

### 1. 接入层 (API Layer)

**位置**: `cmd/spx-backend/`

**组件**:
- `post_image.yap` - 图片生成API路由
- `post_image_svg.yap` - SVG生成API路由  
- `post_images_recommend.yap` - 图片推荐API路由

**职责**:
- HTTP请求接入
- 路由分发
- 基础参数验证
- 响应格式化

**API端点**:
```
POST /images          → 生成图片(元数据)
POST /images/svg      → 生成SVG(内容)
POST /images/recommend → 推荐图片
```

### 2. 应用层 (Application Layer)

**位置**: `internal/controller/`

#### AI生图服务 (`svg.go`)

**核心类**:
- `GenerateSVGParams` - 生成参数
- `SVGResponse` - SVG响应
- `ImageResponse` - 图片元数据响应

**核心方法**:
- `GenerateSVG(ctx, params)` - 生成SVG并返回内容
- `GenerateImage(ctx, params)` - 生成图片并返回元数据
- `getSVGContent(ctx, url)` - 获取SVG内容
- `callVectorService(ctx, id, url, content)` - 调用向量服务

**业务流程**:
```
参数验证 → 主题应用 → Provider调用 → 内容下载 → Kodo存储 → 数据库入库 → 向量化
```

#### 图片推荐服务 (`image_recommend.go`)

**核心类**:
- `ImageRecommendParams` - 推荐参数
- `ImageRecommendResult` - 推荐结果
- `RecommendedImageResult` - 单个推荐项

**核心方法**:
- `RecommendImages(ctx, params)` - 混合推荐主流程
- `callAlgorithmService(ctx, text, topK)` - 调用算法服务
- `generateAISVGs(ctx, text, provider, count, startRank)` - 并发生成AI图片

**业务流程**:
```
语义搜索 → 数据库匹配 → 不足检测 → AI生成补全 → 结果聚合 → 排序返回
```

### 3. 领域层 (Domain Layer)

**位置**: `internal/svggen/`

#### 核心组件

**ServiceManager** (`svggen.go`):
```go
type ServiceManager struct {
    svgioService     ProviderService    // SVGIO提供商
    recraftService   ProviderService    // Recraft提供商  
    openaiService    ProviderService    // OpenAI提供商
    translateService TranslateService   // 翻译服务
}
```

**Provider实现**:
- `svgio.go` - SVGIO专业SVG生成
- `recraft.go` - Recraft高质量矢量图
- `openai.go` - OpenAI通用AI绘图

**领域模型** (`types.go`):
- `Provider` - 提供商枚举
- `GenerateRequest` - 生成请求
- `ImageResponse` - 图片响应

#### 业务规则

**Provider选择策略**:
- 主题优先: 根据主题自动选择最佳Provider
- 翻译策略: SVGIO需要翻译，Recraft/OpenAI原生中文支持
- 降级机制: Provider不可用时自动fallback

**推荐策略**:
- 搜索优先: 语义匹配现有资源
- 智能补全: 不足时AI生成
- 相似度分配: 搜索真实度，生成递减分配

### 4. 基础设施层 (Infrastructure Layer)

#### 数据持久化 (`internal/model/`)

**数据模型**:
- `AIResource` - AI资源表
- `Label` - 标签表
- `ResourceLabel` - 资源标签关系表
- `ResourceUsageStats` - 使用统计表

**表结构**:
```sql
-- AI资源表
aiResource: id, url, created_at, updated_at, deleted_at

-- 使用统计表  
resource_usage_stats: ai_resource_id, view_count, selection_count, last_used_at
```

#### 存储服务

**Kodo存储** (`internal/controller/controller.go`):
```go
type kodoClient struct {
    cred         *qiniuAuth.Credentials
    bucket       string
    bucketRegion string  
    baseUrl      string
}
```

**功能**:
- 文件上传
- URL生成
- 访问控制

#### 配置管理 (`internal/config/`)
- Provider配置
- 存储配置
- 服务地址配置

#### 日志服务 (`internal/log/`)
- 请求链路追踪
- 错误日志记录
- 性能监控

## 外部依赖服务

### 核心外部服务
```
spx-algorithm (语义搜索)  ←→  应用层
     ↓
OpenAI/Recraft/SVGIO     ←→  领域层  
     ↓
Kodo云存储              ←→  基础设施层
     ↓
MySQL数据库             ←→  基础设施层
     ↓  
向量数据库              ←→  基础设施层
```

### 服务通信

**spx-algorithm服务**:
- 地址: `http://100.100.35.128:5000`
- 接口: `POST /api/search/resource`
- 用途: 图片语义搜索

**AI生图Provider**:
- SVGIO: 专业SVG生成服务
- Recraft: 高质量矢量图服务
- OpenAI: 通用AI绘图服务

**向量服务**:
- 地址: `http://100.100.35.128:5000`
- 接口: `POST /api/vector/add`
- 用途: 图片向量化存储

## 数据流向

### AI生图服务数据流
```
用户请求 → 接入层 → 应用层 → 领域层 → Provider服务
                ↓           ↓
            参数验证     服务调用
                ↓           ↓  
            主题应用     内容下载
                ↓           ↓
            Kodo存储 ← 基础设施层
                ↓
            数据库入库
                ↓
            向量化存储
                ↓
            响应返回
```

### 图片推荐服务数据流
```
用户请求 → 接入层 → 应用层 → spx-algorithm
                ↓           ↓
            参数验证     语义搜索
                ↓           ↓
            数据库查询 ← 基础设施层
                ↓
            不足检测
                ↓
            AI生成 → 领域层 → Provider服务
                ↓
            结果聚合
                ↓  
            响应返回
```

## 设计原则与特点

### 分层原则
1. **单向依赖**: 上层依赖下层，下层不依赖上层
2. **职责分离**: 每层专注自己的职责
3. **接口隔离**: 通过接口解耦具体实现
4. **开闭原则**: 对扩展开放，对修改关闭

### 架构特点
1. **策略模式**: 多Provider可插拔设计
2. **混合推荐**: 搜索+生成保证结果完整性
3. **异步处理**: 并发生成提升性能
4. **服务降级**: 外部服务失败时的fallback
5. **统一接口**: 所有资源统一KodoURL访问

### 扩展点
1. **新增Provider**: 实现ProviderService接口
2. **新增推荐策略**: 扩展推荐算法
3. **新增存储**: 抽象存储接口
4. **新增缓存**: 添加缓存层

## 监控与运维

### 关键监控指标
- **生图服务**: 成功率、响应时间、Provider分布
- **推荐服务**: 搜索命中率、生成补足率、混合比例
- **存储服务**: 上传成功率、存储空间使用率
- **外部依赖**: 算法服务可用性、Provider服务状态

### 日志策略
- **请求级别**: 每个请求全链路日志
- **组件级别**: 各组件关键操作日志  
- **错误级别**: 异常和降级日志
- **性能级别**: 耗时统计和瓶颈分析

---

## 文档版本信息

- **版本**: v1.0
- **创建时间**: 2025-09-01
- **维护者**: SPX-Backend团队
- **更新周期**: 架构变更时同步更新

---

## 使用说明

本文档为分层架构设计文档，建议：

1. **架构师**: 用于系统设计和技术选型
2. **开发人员**: 了解代码组织和职责划分
3. **运维人员**: 理解系统依赖和监控要点
4. **新人onboarding**: 快速理解系统架构

如需制作分层图，可参考本文档的层级结构和组件关系进行可视化设计。
