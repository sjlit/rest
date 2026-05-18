# schema-ui CSS 样式补全设计

## 背景

`packages/schema-ui/` 项目中大量 Vue 组件使用了自定义 `class`，但项目下没有任何对应的 CSS/SCSS 文件。这导致组件在实际渲染时缺少必要的布局、间距和视觉层次，完全依赖 Element Plus 的默认样式。

本次设计的目标是为这些自定义 class 提供一套**轻量、精致、与 Element Plus 原生风格协调**的样式体系。

## 设计原则

1. **轻量补全**：只做布局、间距、颜色层面的补充，不覆盖 Element Plus 组件本身的样式（如 `el-table`、`el-form-item` 等保持原样）。
2. **主题自适应**：样式值优先使用 Element Plus 的 CSS 变量（`--el-*`），换主题时自动协调。
3. **工程精致**：建立有规律的间距系统、统一的过渡动画、微妙的阴影层次。
4. **模块化组织**：每个组件对应独立的样式文件，由统一入口导出。
5. **构建友好**：Vite lib 模式自动提取 CSS，同时输出独立的 `dist/schema-ui.css`，使用方可选择是否引入。

## 文件结构

```
src/
  styles/
    index.css        # 统一入口，导入所有子模块
    variables.css    # schema-ui 自有变量（fallback 到 Element Plus）
    page.css         # SchemaPage 布局样式
    grid.css         # SchemaGrid（含移动端折叠面板）
    form.css         # SchemaForm 布局样式
    cell.css         # Cell 标签/文本样式
```

## CSS 变量体系（variables.css）

建立系统的 spacing scale 和语义化 token：

```css
:root {
  /* Spacing Scale */
  --su-space-1: 4px;
  --su-space-2: 8px;
  --su-space-3: 12px;
  --su-space-4: 16px;
  --su-space-5: 24px;
  --su-space-6: 32px;

  /* Semantic tokens —— 全部 fallback 到 Element Plus */
  --su-gap: var(--su-space-3);
  --su-padding: var(--su-space-4);
  --su-radius: var(--el-border-radius-base, 4px);
  --su-radius-sm: var(--el-border-radius-small, 2px);

  /* Colors */
  --su-bg: var(--el-bg-color, #ffffff);
  --su-bg-page: var(--el-bg-color-page, #f2f3f5);
  --su-text-primary: var(--el-text-color-primary, #303133);
  --su-text-regular: var(--el-text-color-regular, #606266);
  --su-text-secondary: var(--el-text-color-secondary, #909399);
  --su-border: var(--el-border-color, #dcdfe6);
  --su-border-light: var(--el-border-color-lighter, #ebeef5);

  /* Elevation —— 基于文字色的半透明阴影，深浅主题均有效 */
  --su-shadow-color: color-mix(in srgb, var(--su-text-primary) 4%, transparent);
  --su-shadow-color-light: color-mix(in srgb, var(--su-text-primary) 2%, transparent);
  --su-shadow: 0 1px 2px var(--su-shadow-color), 0 1px 6px var(--su-shadow-color-light);
  --su-shadow-hover: 0 2px 8px color-mix(in srgb, var(--su-text-primary) 6%, transparent),
    0 2px 12px color-mix(in srgb, var(--su-text-primary) 4%, transparent);

  /* Motion —— 按属性分层 */
  --su-transition-fast: opacity 0.15s cubic-bezier(0.4, 0, 0.2, 1);
  --su-transition-base: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
```

## 各组件样式设计

### SchemaPage（page.css）

涉及 class：`.schema-page`, `.schema-page-header`, `.header-left`, `.header-right`, `.schema-page-body`, `.schema-page-search`, `.schema-page-toolbar`, `.schema-page-grid`

```css
.schema-page {
  display: flex;
  flex-direction: column;
  gap: var(--su-gap);
  padding: var(--su-space-3);
  background: var(--su-bg-page);
  min-height: 100%;
}

.schema-page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  row-gap: var(--su-space-2);
  padding: var(--su-padding) var(--su-space-5);
  background: var(--su-bg);
  border-radius: var(--su-radius);
  border: 1px solid var(--su-border-light);
  box-shadow: var(--su-shadow);
  transition: var(--su-transition-base);
}

.schema-page-header:hover {
  box-shadow: var(--su-shadow-hover);
}

.schema-page-header:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}

.header-left h3 {
  margin: 0;
  font-size: var(--el-font-size-large, 16px);
  font-weight: 600;
  color: var(--su-text-primary);
  line-height: 1.4;
}

.header-right {
  display: flex;
  gap: var(--su-space-2);
  align-items: center;
}

.schema-page-body {
  display: flex;
  flex-direction: column;
  gap: var(--su-gap);
}

.schema-page-search,
.schema-page-grid {
  padding: var(--su-padding) var(--su-space-5);
  background: var(--su-bg);
  border-radius: var(--su-radius);
  border: 1px solid var(--su-border-light);
  box-shadow: var(--su-shadow);
  transition: var(--su-transition-base);
}

/* Loading mask 圆角适配 */
.schema-page-search > .el-loading-mask,
.schema-page-grid > .el-loading-mask {
  border-radius: var(--su-radius);
}

.schema-page-toolbar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding: var(--su-space-2) var(--su-space-4);
}

/* 分页区域使用 wrapper class 控制布局，不直接覆盖 el-pagination */
.schema-page-pagination {
  margin-top: var(--su-gap);
  padding-top: var(--su-space-3);
  border-top: 1px solid var(--su-border-light);
  display: flex;
  justify-content: flex-end;
}

/* Empty 状态在卡片内的居中协调 */
.schema-page-grid .el-empty {
  padding: var(--su-space-6) 0;
}
```

### SchemaForm（form.css）

涉及 class：`.schema-form`, `.schema-form-actions`

```css
.schema-form {
  padding: var(--su-padding);
}

.schema-form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--su-space-2);
  margin-top: var(--su-space-4);
  padding-top: var(--su-space-3);
  border-top: 1px solid var(--su-border-light);
}
```

### SchemaGrid（grid.css）

涉及 class：`.schema-grid-actions`, `.mobile-primary-label`, `.mobile-actions`, `.mobile-preview-row`, `.mobile-preview-label`, `.mobile-preview-value`

```css
.schema-grid-actions {
  display: flex;
  gap: var(--su-space-2);
  justify-content: center;
  flex-wrap: wrap;
  align-items: center;
}

.mobile-primary-label {
  flex: 1;
  font-weight: 600;
  color: var(--su-text-primary);
  font-size: var(--el-font-size-base, 14px);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: var(--su-space-3);
}

.mobile-actions {
  display: flex;
  gap: var(--su-space-1);
  flex-shrink: 0;
}

.mobile-preview-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: var(--su-space-2) 0;
  border-bottom: 1px solid var(--su-border-light);
  gap: var(--su-space-3);
}

.mobile-preview-row:last-child {
  border-bottom: none;
}

.mobile-preview-label {
  color: var(--su-text-secondary);
  font-size: var(--el-font-size-small, 12px);
  flex-shrink: 0;
  min-width: 60px;
  max-width: 40%;
}

.mobile-preview-value {
  color: var(--su-text-regular);
  text-align: right;
  word-break: break-word;
  flex: 1;
  font-size: var(--el-font-size-base, 14px);
}

/* 极小屏幕：label/value 上下堆叠，避免拥挤 */
@media (max-width: 375px) {
  .mobile-preview-row {
    flex-direction: column;
    align-items: stretch;
    gap: var(--su-space-1);
  }

  .mobile-preview-label {
    max-width: none;
    min-width: auto;
  }

  .mobile-preview-value {
    text-align: left;
  }
}
```

### Cell（cell.css）

涉及 class：`.schema-cell`, `.schema-cell-tag`

```css
.schema-cell {
  color: var(--su-text-regular);
  word-break: break-word;
  line-height: 1.5;
}

.schema-cell-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: var(--su-radius-sm);
  font-size: var(--el-font-size-small, 12px);
  font-weight: 500;
  line-height: 1.4;
  transition: var(--su-transition-base);
  cursor: default;
  user-select: none;
}

.schema-cell-tag:hover {
  opacity: 0.85;
}

.schema-cell-tag:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}
```

## 统一入口（index.css）

```css
@import './variables.css';
@import './page.css';
@import './grid.css';
@import './form.css';
@import './cell.css';
```

## 组件引入方式

每个 Vue 组件在 `<script>` 顶部导入自己的样式文件：

- `SchemaPage.vue` → `import '../styles/page.css'`
- `SchemaGrid.vue` → `import '../styles/grid.css'`
- `SchemaForm.vue` → `import '../styles/form.css'`
- `Cell.vue` → `import '../styles/cell.css'`

Vite lib 模式构建时会自动提取所有 CSS 到 `dist/schema-ui.css`。使用方既可以：

1. 通过 npm 引入时 CSS 随 JS 自动注入（Vite/webpack 默认行为）
2. 通过 CDN 单独链入 `dist/schema-ui.css`

## 关键设计决策

| 决策 | 说明 |
|------|------|
| 独立 CSS 文件 | 组件各自 import 自己的样式，保持模块化；Vite 自动合并提取 |
| Element Plus 变量 fallback | 所有颜色、圆角、字号都 fallback 到 `--el-*`，保证主题一致性 |
| 自适应阴影 | 使用 `color-mix(in srgb, var(--su-text-primary) X%, transparent)` 生成阴影，深浅主题均有效 |
| 过渡分层 | `opacity` 用 `0.15s`，其他属性用 `0.2s`，统一 `cubic-bezier(0.4, 0, 0.2, 1)` |
| 分页 wrapper class | 用 `.schema-page-pagination` 包裹 pagination 控制布局，不直接覆盖 `.el-pagination` |
| loading mask 圆角 | `.schema-page-search > .el-loading-mask` 和 `.schema-page-grid > .el-loading-mask` 继承卡片圆角 |
| 移动端响应式 | 默认左右排列，`< 375px` 时 label/value 上下堆叠 |
| focus-visible | 可交互元素（header、cell-tag）提供 `outline: 2px solid var(--el-color-primary)` |
| 表单/分页区顶边框 | 用细线分隔区域，形成清晰的段落感 |
| empty 状态协调 | `.schema-page-grid .el-empty { padding: var(--su-space-6) 0; }` 提升视觉平衡 |
| header 防溢出 | `flex-wrap: wrap` + `row-gap` 防止右侧按钮过多时挤压标题 |
| cell-tag hover | 使用 `opacity: 0.85` 替代 `brightness`，避免深浅主题效果相反 |
