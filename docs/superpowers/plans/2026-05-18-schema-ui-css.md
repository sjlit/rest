# schema-ui CSS 样式补全实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `packages/schema-ui` 中所有自定义 class 创建对应的模块化 CSS 文件，并修改 Vue 组件引入样式，使构建产物输出独立的 `dist/schema-ui.css`。

**Architecture:** 每个组件对应一个独立的样式文件（`src/styles/*.css`），组件在 `<script>` 中 `import` 自己的样式。Vite lib 模式自动提取所有 CSS 为单独产物。样式使用 Element Plus CSS 变量 fallback，并建立 schema-ui 自有的 spacing/elevation/motion token 体系。

**Tech Stack:** Vue 3, Element Plus, Vite (lib mode), CSS custom properties

---

## 文件结构

```
src/
  styles/
    index.css          # 统一入口
    variables.css      # spacing / color / elevation / motion token
    page.css           # SchemaPage
    grid.css           # SchemaGrid (含移动端折叠面板)
    form.css           # SchemaForm
    cell.css           # Cell 标签/文本
  ui/
    SchemaPage.vue     # + import '../styles/page.css', pagination wrapper
    SchemaGrid.vue     # + import '../styles/grid.css'
    SchemaForm.vue     # + import '../styles/form.css'
    parts/Cell.vue     # + import '../styles/cell.css'
```

---

### Task 1: 创建 CSS 变量文件

**Files:**
- Create: `src/styles/variables.css`

- [ ] **Step 1: 编写 variables.css**

```css
:root {
  --su-space-1: 4px;
  --su-space-2: 8px;
  --su-space-3: 12px;
  --su-space-4: 16px;
  --su-space-5: 24px;
  --su-space-6: 32px;

  --su-gap: var(--su-space-3);
  --su-padding: var(--su-space-4);
  --su-radius: var(--el-border-radius-base, 4px);
  --su-radius-sm: var(--el-border-radius-small, 2px);

  --su-bg: var(--el-bg-color, #ffffff);
  --su-bg-page: var(--el-bg-color-page, #f2f3f5);
  --su-text-primary: var(--el-text-color-primary, #303133);
  --su-text-regular: var(--el-text-color-regular, #606266);
  --su-text-secondary: var(--el-text-color-secondary, #909399);
  --su-border: var(--el-border-color, #dcdfe6);
  --su-border-light: var(--el-border-color-lighter, #ebeef5);

  --su-shadow-color: color-mix(in srgb, var(--su-text-primary) 4%, transparent);
  --su-shadow-color-light: color-mix(in srgb, var(--su-text-primary) 2%, transparent);
  --su-shadow: 0 1px 2px var(--su-shadow-color), 0 1px 6px var(--su-shadow-color-light);
  --su-shadow-hover: 0 2px 8px color-mix(in srgb, var(--su-text-primary) 6%, transparent),
    0 2px 12px color-mix(in srgb, var(--su-text-primary) 4%, transparent);

  --su-transition-fast: opacity 0.15s cubic-bezier(0.4, 0, 0.2, 1);
  --su-transition-base: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
```

- [ ] **Step 2: Commit**

```bash
git add src/styles/variables.css
git commit -m "feat(schema-ui): add CSS variables with spacing, elevation and motion tokens"
```

---

### Task 2: 创建 page.css 并修改 SchemaPage.vue

**Files:**
- Create: `src/styles/page.css`
- Modify: `src/ui/SchemaPage.vue`

- [ ] **Step 1: 编写 page.css**

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

.schema-page-pagination {
  margin-top: var(--su-gap);
  padding-top: var(--su-space-3);
  border-top: 1px solid var(--su-border-light);
  display: flex;
  justify-content: flex-end;
}

.schema-page-grid .el-empty {
  padding: var(--su-space-6) 0;
}
```

- [ ] **Step 2: 修改 SchemaPage.vue —— 在 script 顶部添加 CSS import**

在 `src/ui/SchemaPage.vue` 的 `<script setup>` 顶部（import 语句区域）加入：

```ts
import '../styles/page.css'
```

- [ ] **Step 3: 修改 SchemaPage.vue —— 为 pagination 添加 wrapper div**

将第 47-48 行：

```html
<el-pagination v-if="showPagination" :page-size="pagination.size" :total="pagination.totalCount"
  :current-page="pagination.index" layout="total, prev, pager, next" @current-change="handlePageChange" />
```

改为：

```html
<div v-if="showPagination" class="schema-page-pagination">
  <el-pagination :page-size="pagination.size" :total="pagination.totalCount"
    :current-page="pagination.index" layout="total, prev, pager, next" @current-change="handlePageChange" />
</div>
```

- [ ] **Step 4: Commit**

```bash
git add src/styles/page.css src/ui/SchemaPage.vue
git commit -m "feat(schema-ui): add SchemaPage styles with adaptive shadows and pagination wrapper"
```

---

### Task 3: 创建 grid.css 并修改 SchemaGrid.vue

**Files:**
- Create: `src/styles/grid.css`
- Modify: `src/ui/SchemaGrid.vue`

- [ ] **Step 1: 编写 grid.css**

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

- [ ] **Step 2: 修改 SchemaGrid.vue —— 在 script 顶部添加 CSS import**

在 `src/ui/SchemaGrid.vue` 的 `<script setup>` 顶部加入：

```ts
import '../styles/grid.css'
```

- [ ] **Step 3: Commit**

```bash
git add src/styles/grid.css src/ui/SchemaGrid.vue
git commit -m "feat(schema-ui): add SchemaGrid styles with mobile responsive breakpoint"
```

---

### Task 4: 创建 form.css 并修改 SchemaForm.vue

**Files:**
- Create: `src/styles/form.css`
- Modify: `src/ui/SchemaForm.vue`

- [ ] **Step 1: 编写 form.css**

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

- [ ] **Step 2: 修改 SchemaForm.vue —— 在 script 顶部添加 CSS import**

在 `src/ui/SchemaForm.vue` 的 `<script setup>` 顶部加入：

```ts
import '../styles/form.css'
```

- [ ] **Step 3: Commit**

```bash
git add src/styles/form.css src/ui/SchemaForm.vue
git commit -m "feat(schema-ui): add SchemaForm action area styles"
```

---

### Task 5: 创建 cell.css 并修改 Cell.vue

**Files:**
- Create: `src/styles/cell.css`
- Modify: `src/ui/parts/Cell.vue`

- [ ] **Step 1: 编写 cell.css**

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

- [ ] **Step 2: 修改 Cell.vue —— 在 script 顶部添加 CSS import**

在 `src/ui/parts/Cell.vue` 的 `<script setup>` 顶部加入：

```ts
import '../styles/cell.css'
```

- [ ] **Step 3: Commit**

```bash
git add src/styles/cell.css src/ui/parts/Cell.vue
git commit -m "feat(schema-ui): add Cell tag and text styles with focus-visible"
```

---

### Task 6: 创建 index.css 统一入口

**Files:**
- Create: `src/styles/index.css`

- [ ] **Step 1: 编写 index.css**

```css
@import './variables.css';
@import './page.css';
@import './grid.css';
@import './form.css';
@import './cell.css';
```

- [ ] **Step 2: Commit**

```bash
git add src/styles/index.css
git commit -m "feat(schema-ui): add unified CSS entry point"
```

---

### Task 7: 构建验证 CSS 产物

**Files:**
- Verify: `dist/schema-ui.css` 存在且内容正确

- [ ] **Step 1: 运行构建**

```bash
cd packages/schema-ui && npm run build
```

- [ ] **Step 2: 验证 CSS 产物存在**

```bash
ls -la packages/schema-ui/dist/schema-ui.css
```

Expected: 文件存在，大小 > 0

- [ ] **Step 3: 验证 CSS 产物内容包含关键 class**

```bash
grep -c "schema-page" packages/schema-ui/dist/schema-ui.css
grep -c "schema-grid-actions" packages/schema-ui/dist/schema-ui.css
grep -c "schema-form-actions" packages/schema-ui/dist/schema-ui.css
grep -c "schema-cell-tag" packages/schema-ui/dist/schema-ui.css
```

Expected: 每条输出都 ≥ 1

- [ ] **Step 4: Commit 构建产物（若产物在版本控制内）或记录验证结果**

如果 `dist/` 在 `.gitignore` 中，则无需提交产物。运行 `git status` 确认只有源码文件有变更。

```bash
git status
```

Expected: 无未提交的源码变更，dist/ 被忽略。

- [ ] **Step 5: 最终提交（若验证通过）**

```bash
git commit --allow-empty -m "chore(schema-ui): verify CSS build output"
```

---

## Self-Review

### 1. Spec coverage

| Spec 需求 | 对应任务 |
|-----------|----------|
| spacing scale（4/8/12/16/24/32） | Task 1 |
| 语义化 token（gap/padding/radius/bg/color） | Task 1 |
| 自适应阴影（color-mix） | Task 1 |
| 过渡分层（fast/base） | Task 1 |
| SchemaPage 布局（header/body/search/toolbar/grid） | Task 2 |
| loading mask 圆角覆盖 | Task 2 |
| pagination wrapper class | Task 2 |
| empty 状态 padding 协调 | Task 2 |
| header flex-wrap 防溢出 | Task 2 |
| focus-visible 状态 | Task 2, Task 5 |
| SchemaForm 操作区 | Task 4 |
| SchemaGrid 操作列 + 移动端折叠面板 | Task 3 |
| 移动端 <375px 响应式 | Task 3 |
| Cell 标签/文本样式 | Task 5 |
| cell-tag hover 改用 opacity | Task 5 |
| 统一入口 index.css | Task 6 |
| 构建产物验证 | Task 7 |

**无遗漏。**

### 2. Placeholder scan

- 无 "TBD"/"TODO"/"implement later"
- 无 "Add appropriate error handling" 等模糊描述
- 无 "Similar to Task N" 的引用
- 每个步骤包含实际代码或精确命令

### 3. Type 一致性

- 所有 CSS 变量名在 Task 1 定义后，后续任务使用一致（`--su-space-*`, `--su-transition-base` 等）
- 文件路径在全文中一致

---

## 执行方式选择

Plan complete and saved to `docs/superpowers/plans/2026-05-18-schema-ui-css.md`.

**Two execution options:**

**1. Subagent-Driven (recommended)** - Dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session, batch execution with checkpoints

Which approach do you prefer?
