# @nobla/rest-ui 发布指南

前端组件库 `@nobla/rest-ui`（位于 `packages/rest-ui/`）的 npm 发布操作手册。

- **包名**：`@nobla/rest-ui`（scoped，发布到公共 npm registry）
- **仓库位置**：`packages/rest-ui/`
- **Registry**：`https://registry.npmjs.org/`
- **技术栈**：Vue 3 + Element Plus + TypeScript，Vite lib 模式构建
- **Peer Dependencies**（不打包，消费方自行安装）：`vue@^3.3.0`、`element-plus@^2.12.0`、`@element-plus/icons-vue@^2.3.0`

发布流程：`登录 → 构建 → 检查发布物 → 发布 → 验证`。

---

## 1. 前置条件

### 1.1 npm 账号

需要拥有 npm 账号，且该账号属于 `@nobla` 组织（scope 归属校验在发布时进行）：

```bash
# 如果 @nobla 组织尚未创建（或账号不属于该组织）：
npm org create nobla
```

### 1.2 package.json 配置（已就绪）

| 字段 | 值 | 说明 |
|---|---|---|
| `name` | `@nobla/rest-ui` | scope 名与 Go 模块组织（git.nobla.cn）对齐 |
| `version` | `1.0.0` | 语义化版本，发版时递增 |
| `license` | `MIT` | 如有公司许可证约定，发布前替换 |
| `files` | `["dist"]` | 只发布构建产物；`package.json` / `README.md` 自动包含 |
| `publishConfig.access` | `public` | **必需**：scope 包默认受限/私有，没有此项发布会被拒 |
| `main` / `module` / `types` / `exports` | `dist/rest-ui.*` | 入口与产物文件名对应 |

### 1.3 构建产物

`dist/` 目录包含：

- `rest-ui.es.js` — ESM 格式
- `rest-ui.cjs` — CJS 格式
- `style.css` — 组件样式（构建时自动随 JS 注入）
- `index.d.ts` + `.d.ts.map` — TypeScript 类型声明

`dist/` 已被 `.gitignore` 排除，发布物以本地构建结果为准，**发布前必须重新构建**。

---

## 2. 登录 npm

`npm login` 是交互式命令，请在终端（或 Claude Code 会话中通过 `! npm login`）执行：

```bash
npm login
```

验证登录状态：

```bash
npm whoami
```

---

## 3. 构建

```bash
cd packages/rest-ui
npm install        # 首次或依赖变更后
npm run build      # 产物输出到 dist/
```

> 本地开发用 `npm run dev`（watch 构建）；发版用 `npm run build`（一次性构建）。

---

## 4. 检查发布物（推荐）

发布前确认 tarball 内容符合预期：

```bash
npm pack --dry-run
```

预期输出示例：

```
npm notice name: @nobla/rest-ui
npm notice version: 1.0.0
npm notice filename: nobla-rest-ui-1.0.0.tgz
npm notice total files: 45
```

检查要点：

- 只包含 `dist/` + `package.json` + `README.md`，**不应包含** `src/`、`node_modules/`、`*.log`
- 类型声明文件（`index.d.ts`）已包含

---

## 5. 发布

```bash
npm publish
```

> `publishConfig.access: public` 已配置，无需追加 `--access public`。

---

## 6. 发布后验证

```bash
# 查看包信息（版本、README、文件列表）
npm view @nobla/rest-ui

# 在消费项目中真实安装一次
npm install @nobla/rest-ui

# 安装 peerDependencies
npm install vue@^3.3.0 element-plus@^2.12.0 @element-plus/icons-vue@^2.3.0
```

---

## 7. 版本管理

### 7.1 语义化版本

| 场景 | 命令 | 示例 |
|---|---|---|
| 修复 bug | `npm version patch` | 1.0.0 → 1.0.1 |
| 新增功能（向后兼容） | `npm version minor` | 1.0.0 → 1.1.0 |
| 破坏性变更 | `npm version major` | 1.0.0 → 2.0.0 |

> `npm version` 默认会在仓库根创建 git tag + commit；不想要自动 tag 时追加 `--no-git-tag-version`，手动 `git tag v1.0.1 && git push --tags`。

### 7.2 预发布版本

```bash
npm version 1.1.0-beta.0
npm publish --tag beta
```

消费方通过 tag 安装：

```bash
npm install @nobla/rest-ui@beta
```

`npm publish --tag beta` 不会覆盖 `latest`，正式版发布后 beta 用户需手动升级。

### 7.3 发布顺序建议

1. `npm version <patch|minor|major>`（版本号 + git tag）
2. `npm run build`
3. `npm pack --dry-run`
4. `npm publish`
5. `npm view @nobla/rest-ui` 验证
6. `git push` 推送代码与 tag

---

## 8. 旧包迁移（如适用）

若 `@ace/schema-ui` 之前已发布过，向使用方发送迁移提示：

```bash
npm deprecate @ace/schema-ui "已更名为 @nobla/rest-ui，请迁移"
```

> 注意：`@ace/schema-ui` 与 `@nobla/rest-ui` 是两个独立的包；重命名不会自动转移下载量或版本号。

---

## 9. CI 自动发布（可选）

使用 GitHub Actions，在 tag 推送时自动构建并发布：

1. 生成发布 token（需登录 npm）：

   ```bash
   npm token create --read-only --publish
   ```

2. 将 token 配置为仓库 Secret（如 `NPM_TOKEN`）。

3. 工作流示例 `.github/workflows/publish.yml`：

   ```yaml
   name: Publish @nobla/rest-ui

   on:
     push:
       tags: ['v*']

   jobs:
     publish:
       runs-on: ubuntu-latest
       defaults:
         run:
           working-directory: packages/rest-ui
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-node@v4
           with:
             node-version: 22
             registry-url: https://registry.npmjs.org/
         - run: npm ci
         - run: npm test
         - run: npm run build
         - run: npm publish
           env:
             NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
   ```

   tag 约定：`v1.0.1` 触发正式发布；`v1.1.0-beta.0` 触发 `--tag beta`（需在 workflow 中按 tag 分支处理）。

---

## 10. 常见问题

| 错误 | 原因 | 解决 |
|---|---|---|
| `ENEEDAUTH` / `need auth` | 未登录 | `npm login` 后 `npm whoami` 确认 |
| `E403` / `scope not found` | `@nobla` 组织不存在或账号不在组织内 | `npm org create nobla` 或联系组织管理员加人 |
| `402 Payment Required` | scope 包按私有发布（未配 `publishConfig.access: public`） | 检查 package.json 的 `publishConfig`，或发布时加 `--access public` |
| `E409` / version exists | 版本号已被占用 | 递增版本号后再发布 |
| 发布内容多出 `src/` | `files` 字段缺失或误写 | 确认 `files: ["dist"]` |
| 消费方 `Cannot find module 'element-plus'` | peerDependencies 未安装 | 消费方安装 `vue` / `element-plus` / `@element-plus/icons-vue` |

---

## 11. 回滚

npm 不允许删除已发布版本（会造成消费者缓存解析损坏）。如发布异常：

- **紧急回退**：发布更高版本修复（`npm version patch`）
- **废弃版本**：`npm deprecate @nobla/rest-ui@<version> "此版本有问题，请升级到 <fixed>"`（保留包但标记弃用）
- 仅当包从未被安装（24 小时内且无下载量）可联系 npm 支持移除
