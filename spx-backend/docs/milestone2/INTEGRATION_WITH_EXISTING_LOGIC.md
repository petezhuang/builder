# 现有推荐和生图逻辑集成方案

## 概述

本文档详细说明如何将用户画像和预加载功能与现有的图片推荐(`POST /images/recommend`)和AI生图(`POST /image/svg`)逻辑深度集成，确保现有API保持完全兼容的同时，增加个性化能力。

## 核心集成策略

### 1. API兼容性保证
- **现有接口不变**: 保持`POST /images/recommend`和`POST /image/svg`的请求格式完全不变
- **响应格式扩展**: 在现有响应基础上增加个性化字段，向下兼容
- **性能要求**: 新功能不能显著影响现有接口的响应时间

### 2. 渐进式增强策略
- **Phase 1**: 后台静默收集用户行为数据
- **Phase 2**: 增加个性化评分和排序
- **Phase 3**: 启用预加载缓存优化
- **Phase 4**: 全面个性化推荐

## 详细集成实现

### 一、增强图片推荐Controller

```go

// EnhancedImageRecommendController 增强的图片推荐控制器
type EnhancedImageRecommendController struct {
	// 现有依赖
	algorithmService AlgorithmService
	resourceService  ResourceService
	svgGenService    SVGGenService
	
	// 新增依赖
	profileService   UserProfileService
	preloadService   PreloadService
	behaviorService  BehaviorTrackingService
}

// RecommendImages 增强的图片推荐接口 - 保持API兼容性
func (ctrl *EnhancedImageRecommendController) RecommendImages(ctx context.Context, params *ImageRecommendParams) (*ImageRecommendResult, error) {
	userID := getUserIDFromContext(ctx)
	startTime := time.Now()
	
	// === Phase 1: 用户画像获取和行为记录 ===
	var userProfile *UserProfile
	if userID != "" {
		// 异步记录搜索行为（不影响主流程性能）
		go ctrl.behaviorService.RecordSearchBehavior(context.Background(), userID, params.Text, params.Provider)
		
		// 获取用户画像（如果存在）
		userProfile, _ = ctrl.profileService.GetUserProfile(ctx, userID)
	}
	
	// === Phase 2: 智能预加载检查 ===
	var preloadResults []*ImageRecommendItem
	if userProfile != nil {
		// 首先尝试从预加载缓存获取
		preloadResults = ctrl.checkPreloadCache(ctx, userProfile, params.Text, params.TopK)
		if len(preloadResults) >= params.TopK {
			// 预加载完全命中，直接返回
			return &ImageRecommendResult{
				Query:        params.Text,
				ResultsCount: len(preloadResults),
				Results:      preloadResults,
				Source:       "preload_cache",
				ResponseTime: time.Since(startTime).Milliseconds(),
			}, nil
		}
	}
	
	// === Phase 3: 增强语义搜索 ===
	searchParams := ctrl.enhanceSearchParams(params, userProfile)
	algResult, err := ctrl.algorithmService.SearchWithPersonalization(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("algorithm service failed: %w", err)
	}
	
	// === Phase 4: 数据库匹配和个性化排序 ===
	foundResults := make([]*ImageRecommendItem, 0, params.TopK)
	
	// 优先使用预加载结果
	foundResults = append(foundResults, preloadResults...)
	
	// 补充语义搜索结果
	for _, algResult := range algResult.Results {
		if len(foundResults) >= params.TopK {
			break
		}
		
		resource, err := ctrl.resourceService.GetResourceByPath(ctx, algResult.ImagePath)
		if err == nil {
			item := &ImageRecommendItem{
				ID:        resource.ID,
				ImagePath: resource.ImagePath,
				Rank:      len(foundResults) + 1,
				Source:    "search",
				Score:     algResult.Score,
			}
			
			// 个性化评分增强
			if userProfile != nil {
				item.PersonalizationScore = ctrl.calculatePersonalizationScore(resource, userProfile)
				item.RecommendationReason = ctrl.generateRecommendationReason(resource, userProfile)
			}
			
			foundResults = append(foundResults, item)
		}
	}
	
	// === Phase 5: AI生成补充（如果需要）===
	if len(foundResults) < params.TopK {
		needed := params.TopK - len(foundResults)
		
		// 根据用户画像优化生成参数
		genParams := ctrl.optimizeGenerationParams(params, userProfile)
		
		generatedResults, err := ctrl.generateAISVGs(ctx, genParams, needed, len(foundResults))
		if err == nil {
			foundResults = append(foundResults, generatedResults...)
		}
	}
	
	// === Phase 6: 最终排序和结果优化 ===
	finalResults := ctrl.personalizedRerank(foundResults, userProfile)
	
	// === Phase 7: 异步更新用户画像 ===
	if userID != "" {
		go ctrl.updateUserProfileAsync(context.Background(), userID, params.Text, finalResults)
	}
	
	return &ImageRecommendResult{
		Query:            params.Text,
		ResultsCount:     len(finalResults),
		Results:          finalResults,
		Source:           "hybrid",
		ResponseTime:     time.Since(startTime).Milliseconds(),
		ProfileContext:   ctrl.buildProfileContext(userProfile),
		CacheHitRate:     float64(len(preloadResults)) / float64(len(finalResults)),
	}, nil
}

// checkPreloadCache 检查预加载缓存
func (ctrl *EnhancedImageRecommendController) checkPreloadCache(ctx context.Context, profile *UserProfile, query string, topK int) []*ImageRecommendItem {
	// 1. 构建缓存查询键
	cacheKey := fmt.Sprintf("preload:%s:%s", profile.GroupID, normalizeQuery(query))
	
	// 2. 从缓存获取预加载结果
	cached, err := ctrl.preloadService.GetCachedResults(ctx, cacheKey)
	if err != nil || len(cached) == 0 {
		return nil
	}
	
	// 3. 转换为推荐结果格式
	results := make([]*ImageRecommendItem, 0, min(len(cached), topK))
	for i, item := range cached {
		if i >= topK {
			break
		}
		
		results = append(results, &ImageRecommendItem{
			ID:                   item.ResourceID,
			ImagePath:            item.ResourcePath,
			Rank:                 i + 1,
			Source:               "preload",
			Score:                item.MatchScore,
			PersonalizationScore: item.PersonalizationScore,
			RecommendationReason: "基于您的使用偏好预选",
		})
	}
	
	return results
}

// enhanceSearchParams 根据用户画像增强搜索参数
func (ctrl *EnhancedImageRecommendController) enhanceSearchParams(params *ImageRecommendParams, profile *UserProfile) *PersonalizedSearchParams {
	enhanced := &PersonalizedSearchParams{
		Query:    params.Text,
		TopK:     params.TopK * 2, // 扩大搜索范围以便个性化排序
		Provider: params.Provider,
	}
	
	if profile != nil {
		// 添加用户偏好的主题权重
		enhanced.ThemeWeights = profile.Preferences.Themes
		
		// 添加用户偏好的内容类型
		enhanced.ContentTypeWeights = profile.Preferences.ContentTypes
		
		// 添加用户群组信息
		enhanced.UserGroup = profile.GroupID
		
		// 历史交互增强
		enhanced.HistoricalPreferences = profile.GetTopPreferences(5)
	}
	
	return enhanced
}

// calculatePersonalizationScore 计算个性化评分
func (ctrl *EnhancedImageRecommendController) calculatePersonalizationScore(resource *AIResource, profile *UserProfile) float64 {
	if profile == nil {
		return 0.5 // 默认中性评分
	}
	
	score := 0.0
	
	// 1. 主题匹配度 (40%)
	if themeWeight, exists := profile.Preferences.Themes[resource.Theme]; exists {
		score += themeWeight * 0.4
	}
	
	// 2. 标签匹配度 (30%)
	labelScore := 0.0
	for _, label := range resource.Labels {
		if weight, exists := profile.Preferences.Labels[label]; exists {
			labelScore += weight
		}
	}
	if len(resource.Labels) > 0 {
		score += (labelScore / float64(len(resource.Labels))) * 0.3
	}
	
	// 3. 历史交互 (20%)
	interactionScore := ctrl.getHistoricalInteractionScore(resource.ID, profile.UserID)
	score += interactionScore * 0.2
	
	// 4. 时间偏好 (10%)
	timeScore := ctrl.getTimePreferenceScore(resource.CreatedAt, profile.UsagePattern.PeakHours)
	score += timeScore * 0.1
	
	return min(score, 1.0)
}

// generateRecommendationReason 生成推荐原因
func (ctrl *EnhancedImageRecommendController) generateRecommendationReason(resource *AIResource, profile *UserProfile) string {
	if profile == nil {
		return "热门推荐"
	}
	
	reasons := []string{}
	
	// 主题匹配
	if weight, exists := profile.Preferences.Themes[resource.Theme]; exists && weight > 0.7 {
		reasons = append(reasons, fmt.Sprintf("符合您喜爱的%s风格", resource.Theme))
	}
	
	// 标签匹配
	matchedLabels := []string{}
	for _, label := range resource.Labels {
		if weight, exists := profile.Preferences.Labels[label]; exists && weight > 0.6 {
			matchedLabels = append(matchedLabels, label)
		}
	}
	if len(matchedLabels) > 0 {
		reasons = append(reasons, fmt.Sprintf("包含您感兴趣的标签: %v", matchedLabels))
	}
	
	// 使用历史
	if ctrl.hasHistoricalInteraction(resource.ID, profile.UserID) {
		reasons = append(reasons, "您之前使用过类似素材")
	}
	
	if len(reasons) == 0 {
		return "为您精选推荐"
	}
	
	return strings.Join(reasons, "，")
}

// optimizeGenerationParams 根据用户画像优化生成参数
func (ctrl *EnhancedImageRecommendController) optimizeGenerationParams(params *ImageRecommendParams, profile *UserProfile) *OptimizedGenParams {
	genParams := &OptimizedGenParams{
		Text:     params.Text,
		Provider: params.Provider,
	}
	
	if profile != nil {
		// 选择用户偏好的主题
		preferredTheme := profile.GetTopTheme()
		if preferredTheme != "" {
			genParams.Theme = preferredTheme
		}
		
		// 根据用户偏好增强prompt
		genParams.EnhancedPrompt = ctrl.enhancePromptWithProfile(params.Text, profile)
		
		// 选择最适合的Provider
		genParams.Provider = ctrl.selectOptimalProvider(profile)
	}
	
	return genParams
}

// personalizedRerank 个性化重排序
func (ctrl *EnhancedImageRecommendController) personalizedRerank(results []*ImageRecommendItem, profile *UserProfile) []*ImageRecommendItem {
	if profile == nil {
		return results
	}
	
	// 按个性化评分重新排序
	sort.Slice(results, func(i, j int) bool {
		scoreI := results[i].Score*0.6 + results[i].PersonalizationScore*0.4
		scoreJ := results[j].Score*0.6 + results[j].PersonalizationScore*0.4
		return scoreI > scoreJ
	})
	
	// 更新排名
	for i, result := range results {
		result.Rank = i + 1
	}
	
	return results
}

// updateUserProfileAsync 异步更新用户画像
func (ctrl *EnhancedImageRecommendController) updateUserProfileAsync(ctx context.Context, userID, query string, results []*ImageRecommendItem) {
	// 提取搜索特征
	features := extractSearchFeatures(query, results)
	
	// 更新用户画像
	err := ctrl.profileService.UpdateProfileWithSearchFeatures(ctx, userID, features)
	if err != nil {
		// 记录错误但不影响主流程
		fmt.Printf("Failed to update user profile: %v", err)
	}
	
	// 触发预加载缓存更新
	ctrl.preloadService.TriggerCacheRefresh(ctx, userID)
}

// 工具函数
func normalizeQuery(query string) string {
	// 查询标准化逻辑
	return strings.ToLower(strings.TrimSpace(query))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func extractSearchFeatures(query string, results []*ImageRecommendItem) *SearchFeatures {
	// 从搜索结果中提取特征用于画像更新
	return &SearchFeatures{
		Query:       query,
		ResultCount: len(results),
		TopThemes:   extractTopThemes(results),
		Timestamp:   time.Now(),
	}
}

func extractTopThemes(results []*ImageRecommendItem) []string {
	// 从结果中提取主要主题
	themes := make(map[string]int)
	for _, result := range results {
		if result.Theme != "" {
			themes[result.Theme]++
		}
	}
	
	var topThemes []string
	for theme := range themes {
		topThemes = append(topThemes, theme)
	}
	
	return topThemes
}
```

### 二、增强SVG生成Controller

```go
// EnhancedSVGController 增强的SVG生成控制器
type EnhancedSVGController struct {
	// 继承现有的SVGGenController
	*SVGGenController
	
	// 新增依赖
	profileService  UserProfileService
	behaviorService BehaviorTrackingService
}

// GenerateSVG 增强的SVG生成接口 - 保持API兼容性
func (ctrl *EnhancedSVGController) GenerateSVG(ctx context.Context, params *GenerateSVGParams) (*GenerateSVGResult, error) {
	userID := getUserIDFromContext(ctx)
	
	// === Phase 1: 获取用户画像 ===
	var userProfile *UserProfile
	if userID != "" {
		userProfile, _ = ctrl.profileService.GetUserProfile(ctx, userID)
	}
	
	// === Phase 2: 基于画像优化生成参数 ===
	optimizedParams := ctrl.optimizeSVGParams(params, userProfile)
	
	// === Phase 3: 调用现有生成逻辑 ===
	result, err := ctrl.SVGGenController.GenerateSVG(ctx, optimizedParams)
	if err != nil {
		return result, err
	}
	
	// === Phase 4: 记录生成行为 ===
	if userID != "" {
		go ctrl.recordGenerationBehavior(context.Background(), userID, params, result)
	}
	
	return result, nil
}

// optimizeSVGParams 根据用户画像优化SVG生成参数
func (ctrl *EnhancedSVGController) optimizeSVGParams(params *GenerateSVGParams, profile *UserProfile) *GenerateSVGParams {
	optimized := *params // 复制原始参数
	
	if profile == nil {
		return &optimized
	}
	
	// 1. 自动选择最适合的主题
	if optimized.Theme == "" {
		preferredTheme := profile.GetTopTheme()
		if preferredTheme != "" {
			optimized.Theme = preferredTheme
		}
	}
	
	// 2. 根据用户偏好选择Provider
	if optimized.Provider == "" {
		optimalProvider := ctrl.selectOptimalProvider(profile)
		optimized.Provider = optimalProvider
	}
	
	// 3. 增强prompt
	optimized.Text = ctrl.enhancePromptWithUserProfile(params.Text, profile)
	
	return &optimized
}

// enhancePromptWithUserProfile 根据用户画像增强prompt
func (ctrl *EnhancedSVGController) enhancePromptWithUserProfile(originalPrompt string, profile *UserProfile) string {
	if profile == nil {
		return originalPrompt
	}
	
	enhancements := []string{}
	
	// 添加用户偏好的风格描述
	topTheme := profile.GetTopTheme()
	if topTheme != "" {
		themeEnhancement := getThemeEnhancement(topTheme)
		enhancements = append(enhancements, themeEnhancement)
	}
	
	// 添加用户偏好的颜色风格
	topColorStyle := profile.GetTopColorStyle()
	if topColorStyle != "" {
		colorEnhancement := getColorEnhancement(topColorStyle)
		enhancements = append(enhancements, colorEnhancement)
	}
	
	// 组合增强prompt
	if len(enhancements) > 0 {
		return originalPrompt + ", " + strings.Join(enhancements, ", ")
	}
	
	return originalPrompt
}

// selectOptimalProvider 选择最适合的Provider
func (ctrl *EnhancedSVGController) selectOptimalProvider(profile *UserProfile) string {
	// 基于用户历史使用统计选择最佳Provider
	providerStats := profile.ProviderUsageStats
	
	// 如果用户有明显偏好
	if providerStats.OpenAI.SatisfactionScore > 0.8 && providerStats.OpenAI.UsageCount > 5 {
		return "openai"
	}
	if providerStats.Recraft.SatisfactionScore > 0.8 && providerStats.Recraft.UsageCount > 5 {
		return "recraft"
	}
	if providerStats.SVGIO.SatisfactionScore > 0.8 && providerStats.SVGIO.UsageCount > 5 {
		return "svgio"
	}
	
	// 根据用户群组的整体统计选择
	groupStats := ctrl.getGroupProviderStats(profile.GroupID)
	return groupStats.GetOptimalProvider()
}

// recordGenerationBehavior 记录用户生成行为
func (ctrl *EnhancedSVGController) recordGenerationBehavior(ctx context.Context, userID string, params *GenerateSVGParams, result *GenerateSVGResult) {
	behavior := &GenerationBehavior{
		UserID:    userID,
		Prompt:    params.Text,
		Theme:     params.Theme,
		Provider:  params.Provider,
		Success:   result.Success,
		Timestamp: time.Now(),
	}
	
	err := ctrl.behaviorService.RecordGeneration(ctx, behavior)
	if err != nil {
		// 记录错误但不影响主流程
		fmt.Printf("Failed to record generation behavior: %v", err)
	}
}
```

### 三、现有Controller的最小化改动

```go
// 现有ImageRecommendController的改动
// 文件: internal/controller/image_recommend.go

func (ctrl *Controller) RecommendImages(ctx context.Context, params *ImageRecommendParams) (*ImageRecommendResult, error) {
	// 保持现有逻辑完全不变...
	
	// 在返回结果前，仅添加一行代码记录用户行为
	defer func() {
		if userID := getUserIDFromContext(ctx); userID != "" {
			// 异步记录，不影响响应时间
			go recordUserBehaviorAsync(userID, params, result)
		}
	}()
	
	// 现有的推荐逻辑保持不变...
	algResult, err := ctrl.callAlgorithmService(ctx, params.Text, params.TopK)
	// ... 其余逻辑完全保持原样
}

// 新增的异步行为记录函数
func recordUserBehaviorAsync(userID string, params *ImageRecommendParams, result *ImageRecommendResult) {
	// 实现异步行为记录逻辑
	behavior := &SearchBehavior{
		UserID:      userID,
		Query:       params.Text,
		ResultCount: result.ResultsCount,
		Timestamp:   time.Now(),
	}
	
	// 发送到消息队列进行异步处理
	behaviorQueue.Send(behavior)
}
```

### 四、数据库集成方案

#### 4.1 扩展现有AIResource表
```sql
-- 在现有ai_resources表基础上添加字段
ALTER TABLE ai_resources ADD COLUMN IF NOT EXISTS personalization_score DECIMAL(3,2) DEFAULT 0.50;
ALTER TABLE ai_resources ADD COLUMN IF NOT EXISTS usage_stats JSONB DEFAULT '{}';
ALTER TABLE ai_resources ADD COLUMN IF NOT EXISTS quality_score DECIMAL(3,2) DEFAULT 0.80;

-- 添加索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_ai_resources_personalization ON ai_resources(personalization_score);
CREATE INDEX IF NOT EXISTS idx_ai_resources_theme_quality ON ai_resources(theme, quality_score);
```

#### 4.2 新增用户画像相关表
```sql
-- 用户画像主表
CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id VARCHAR(50) NOT NULL,
    preferences JSONB NOT NULL DEFAULT '{}',
    behavior_stats JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

-- 用户行为记录表
CREATE TABLE IF NOT EXISTS user_behaviors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    behavior_type VARCHAR(20) NOT NULL, -- 'search', 'download', 'generate', 'rate'
    content JSONB NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 预加载缓存表
CREATE TABLE IF NOT EXISTS preload_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_key VARCHAR(255) NOT NULL,
    user_group VARCHAR(50) NOT NULL,
    resources JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    hit_count INTEGER DEFAULT 0,
    UNIQUE(cache_key)
);

-- 索引优化
CREATE INDEX idx_user_behaviors_user_type ON user_behaviors(user_id, behavior_type);
CREATE INDEX idx_user_behaviors_timestamp ON user_behaviors(timestamp);
CREATE INDEX idx_preload_cache_group ON preload_cache(user_group);
CREATE INDEX idx_preload_cache_expires ON preload_cache(expires_at);
```

### 五、响应格式扩展

#### 5.1 增强的推荐响应格式
```json
{
  "query": "一只可爱的卡通猫",
  "results_count": 8,
  "results": [
    {
      "id": "ai_res_001",
      "image_path": "/ai-resources/cat_001.svg",
      "rank": 1,
      "source": "search",
      
      // 新增个性化字段（向下兼容）
      "personalization": {
        "score": 0.94,
        "reason": "基于您对卡通和可爱风格的偏好",
        "confidence": 0.87
      }
    }
  ],
  
  // 新增整体个性化上下文
  "profile_context": {
    "user_group": "creative_educator",
    "personalization_enabled": true,
    "cache_hit_rate": 0.35
  },
  
  // 新增性能统计
  "performance": {
    "response_time_ms": 156,
    "cache_hits": 3,
    "algorithm_time_ms": 89,
    "personalization_time_ms": 12
  }
}
```

#### 5.2 增强的SVG生成响应
```json
{
  "success": true,
  "data": "<?xml version='1.0'...",
  "headers": {
    "Content-Type": "image/svg+xml"
  },
  
  // 新增个性化生成信息
  "generation_context": {
    "original_prompt": "一只猫",
    "enhanced_prompt": "一只猫, 卡通风格, 明亮色彩",
    "selected_provider": "openai",
    "selection_reason": "基于您的使用偏好自动选择",
    "personalization_applied": true
  }
}
```

### 六、集成时间线

#### Phase 1: 静默数据收集 (1周)
- 在现有API中添加异步行为记录
- 部署用户行为数据收集基础设施
- 不影响现有功能，纯数据收集

#### Phase 2: 基础画像构建 (1-2周)
- 实现用户画像计算算法
- 构建基础的用户分群功能
- 开始生成用户偏好数据

#### Phase 3: 个性化增强 (1-2周)
- 在推荐结果中添加个性化评分
- 实现基于画像的prompt优化
- A/B测试个性化效果

#### Phase 4: 预加载系统 (1-2周)
- 实现图库预匹配算法
- 部署预加载缓存系统
- 优化响应时间和命中率

#### Phase 5: 全面优化 (持续)
- 性能调优和监控
- 算法迭代和改进
- 用户体验优化

### 七、风险控制

#### 7.1 性能风险
- **监控指标**: API响应时间不能超过现有基线的20%
- **降级机制**: 个性化功能故障时自动回退到原始逻辑
- **缓存策略**: 多级缓存确保高频访问的快速响应

#### 7.2 数据质量风险
- **数据验证**: 严格的用户行为数据验证和清洗
- **隐私保护**: 用户数据加密存储，遵循数据保护法规
- **容错设计**: 画像数据缺失时的默认处理策略

#### 7.3 系统稳定性
- **灰度发布**: 按用户比例逐步开启个性化功能
- **监控告警**: 关键指标异常时的实时告警
- **回滚机制**: 快速回滚到原始系统的能力

### 八、测试策略

#### 8.1 兼容性测试
- 确保现有客户端完全兼容新的API响应
- 验证响应时间没有显著退化
- 测试各种边界条件和异常情况

#### 8.2 A/B测试
- 50%用户启用个性化，50%使用原始逻辑
- 对比用户满意度、使用时长、转化率等指标
- 基于测试结果调整个性化算法参数

#### 8.3 压力测试
- 模拟高并发场景下的系统表现
- 验证新功能不会成为性能瓶颈
- 测试数据库和缓存系统的承载能力

这个集成方案确保了在最小化对现有系统影响的前提下，逐步引入用户画像和预加载功能，实现个性化推荐的目标。