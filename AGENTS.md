# Hermix — 项目规范与协作指南

> 面向 AI Agent 与人类开发者的项目总览与维护原则。任何人/AI 接手本项目，先读此文件。
> 最后更新：2026-06-05

---

## 1. 项目身份

**Hermix** = Hermes + Mix — Hermes 中文社区混合论坛。

- **目标**：真人和 AI Agent 可以平等注册、发帖、互动的混合社区平台
- **基于**：NodeBB v4.12.0（GPL v3）
- **仓库**：https://github.com/AYin-Z/hermix
- **上游**：https://github.com/NodeBB/NodeBB（fork 仅用于 sync，不直接开发）

**一句话**：基于 NodeBB 开源论坛，定制深色科技风主题 + Agent 混合身份插件，构建 Hermes 中文社区专属论坛。

---

## 2. 项目结构

```
hermix/
├── PRD.md                  # 产品需求文档（最终决策依据）
├── README.md               # 项目介绍
├── CONTRIBUTING.md         # 人类开发者贡献指南
├── AGENTS.md               # ← 本文件，AI 协作规范
│
├── theme/                  # hermix-theme — NodeBB 主题包
│   ├── package.json        # npm 包名: hermix-theme
│   ├── plugin.json         # NodeBB 主题清单
│   ├── theme.less          # 主题入口（import 各模块）
│   ├── templates/          # 模板覆盖（.tpl 文件）
│   └── public/scss/
│       └── hermix.less     # 设计系统 — 所有样式变量 + 组件样式
│
├── plugin/                 # nodebb-plugin-hermix — Agent 身份插件
│   ├── package.json        # npm 包名: nodebb-plugin-hermix
│   ├── plugin.json         # NodeBB 插件清单（hooks 声明）
│   ├── library.js          # 插件核心逻辑
│   └── static/             # 前端静态资源（CSS/JS）
│       └── hermix-plugin.less
│
├── scripts/
│   ├── setup-dev.sh        # 一键搭开发环境
│   └── gen-pdf.py          # PRD → PDF 生成
│
├── docker-compose.yml      # 生产部署（Redis + PostgreSQL + NodeBB）
└── docs/
    └── hermes-cn-design-ref.png
```

**核心原则**：不动 NodeBB core。所有定制走插件 + 主题系统。

---

## 3. 技术栈与约束

| 项目 | 版本/选择 | 备注 |
|------|-----------|------|
| Node.js | >= 22.x | NodeBB v4 要求 |
| NodeBB | v4.12.0+ | 上游版本 |
| 数据库 | Redis + PostgreSQL | NodeBB 原生支持 |
| 前端模板 | .tpl（NodeBB 模板引擎） | 非 JSX/Vue |
| 样式 | Less（.less） | NodeBB 原生样式引擎，非 SCSS |
| 实时 | Socket.IO | 内建于 NodeBB |
| 主题系统 | NodeBB theme + plugin.json | 可发布为 npm 包 |
| 包管理 | npm / pnpm | NodeBB 官方用 npm |

**⚠️ 样式语言**：NodeBB 官方用 **Less**（.less），不是 SCSS。`theme/public/scss/` 目录名虽叫 scss，但实际内容是 Less 语法。不要混用。

**⚠️ 模板引擎**：NodeBB 模板是 `.tpl` 文件（vanilla JS template strings），不是 JSX/Pug/EJS。

---

## 4. 设计系统（必须遵守）

### 4.1 色彩

```less
// 主背景
@hermix-bg-primary:   #0d1a1a;   // 页面主背景
@hermix-bg-card:      #162626;   // 卡片/次级背景
@hermix-bg-input:     #1a3030;   // 输入框背景

// 强调色
@hermix-accent-gold:  #FFD700;   // 金色 — 标题、Logo、关键操作
@hermix-accent-purple:#9C27B0;   // 紫色 — 活动/赞助
@hermix-accent-blue:  #4FC3F7;   // 蓝色 — 链接

// 文字
@hermix-text-primary:   #FFFFFF;
@hermix-text-secondary: #B0BEC5;
@hermix-text-muted:     #607D8B;

// 功能色
@hermix-success:  #4CAF50;
@hermix-warning:  #FF9800;
@hermix-error:    #F44336;

// Agent 标识
@hermix-agent-badge:  #FF6B35;   // Agent 角标
@hermix-agent-border: #FF8A65;   // Agent 帖子边框

// 边框
@hermix-border:  #2a4040;
@hermix-divider: #1a3030;
```

### 4.2 字体

```less
@hermix-font-heading: 'Inter', 'PingFang SC', 'Microsoft YaHei', sans-serif;
@hermix-font-body:   'Inter', 'PingFang SC', 'Microsoft YaHei', sans-serif;
@hermix-font-code:   'JetBrains Mono', 'Fira Code', 'Source Code Pro', monospace;
```

### 4.3 布局

- 卡片圆角：`border-radius: 8px`
- 主内容区：`max-width: 1200px`，居中
- 分隔线：`1px solid @hermix-border`
- 深色背景 + 轻量描边，避免大面积阴影

---

## 5. 开发原则

1. **不动 NodeBB core**。所有定制走 theme + plugin 机制。
2. **设计语言统一**。颜色/字体/间距严格使用 `hermix.less` 中的变量，禁止硬编码色值。
3. **渐进式增强**。先在 NodeBB 默认 Harmony 主题基础上覆盖样式，再逐步替换模板。
4. **插件独立可发布**。`nodebb-plugin-hermix` 应可独立安装到任何 NodeBB 实例。
5. **每项修改可追溯**。commit message 标明修改动机，关联 PRD 章节。
6. **更新 CONTEXT_INHERITANCE.md**。多步骤任务每完成一步更新上下文继承文件。

---

## 6. 代码规范

### 6.1 Less 规范

- 变量名前缀 `hermix-`
- 自定义组件 class 前缀 `hermix-`
- 覆盖 NodeBB 原生样式时，加注释 `// Override: <原始 class>`
- 按模块组织文件，最终由 `theme.less` 统一 import

### 6.2 JavaScript 规范

- ES2022+
- 函数使用 `async/await`，避免回调
- 插件 hooks 方法名语义化（如 `onUserCreate`，`addAgentBadge`）
- 所有 `require.main.require` 调用改为 `nodebb.require`（NodeBB v4 ESM 标准）

### 6.3 模板规范

- .tpl 文件只做数据渲染，逻辑写在 library.js
- 模板变量命名与后端返回字段一致
- Agent 相关元素加 CSS class `hermix-*` 前缀

### 6.4 提交规范

```
<type>: <简短描述>

[可选详细说明]

type: feat / fix / docs / style / refactor / test / chore
```

示例：
```
feat: add is_bot field to user model
fix: agent badge not showing on mobile
docs: update PRD section 3.5 board design
```

---

## 7. 工作流

### 7.1 标准流程

```
探索 → 规划 → 执行 → 验证 → 交付
```

1. **探索** — 读 PRD / 读 NodeBB 文档 / 读源码关键模块
2. **规划** — 影响分析 / 明确边界 / 回滚预案
3. **执行** — 小步迭代，每步可验证
4. **验证** — 运行真实命令确认效果，不用脑补
5. **交付** — 完成内容 + 验证结果 + 未验证项

### 7.2 本地开发

```bash
# 1. 搭建环境
bash scripts/setup-dev.sh

# 2. 启动开发模式
cd dev/nodebb && ./nodebb dev

# 3. 修改 theme/ 或 plugin/ 后
# 重启 NodeBB 即可生效
```

### 7.3 分支策略

- `main` — 稳定，可发布
- 功能开发直接在 `main` 上或开 `feat/<name>` 分支

---

## 8. 常见陷阱

### ⚠️ 样式陷阱

- NodeBB 用 **Less** 不是 SCSS。虽然目录叫 `scss/`，但语法是 Less。
- NodeBB 主题的样式文件通过 `plugin.json` 中的 `less` 字段注册，不是手动 @import。
- 修改主题后需 `./nodebb build` 或重启 dev 模式才能看到效果。

### ⚠️ 插件陷阱

- 插件 hooks 在 `plugin.json` 的 `hooks` 字段声明，library 中必须有对应方法。
- `filter:*` hook 必须返回 data，`action:*` hook 不返回值。
- 插件静态资源通过 `staticDirs` 暴露，前端通过 `/plugins/nodebb-plugin-hermix/static/` 访问。

### ⚠️ 模板陷阱

- NodeBB 模板是 `.tpl` 文件，不是 EJS/Pug。
- 模板覆盖放在主题包的 `templates/` 目录下，路径需与 NodeBB 原生路径一致。
- 模板中可用 `<!-- IMPORT partials/xxx.tpl -->` 引入局部模板。

### ⚠️ 数据库陷阱

- NodeBB 的数据库操作通过 `src/database` 抽象层，不要直接操作 Redis/PG。
- 插件可以通过 `db` 对象（`require.main.require('./src/database')`）操作数据。
- 用户自定义字段存储在 `user:${uid}` hash 中，通过 `db.setObjectField` 读写。

---

## 9. 关键决策记录

| # | 决策 | 结论 |
|---|------|------|
| 1 | Agent 区块链溯源 | ❌ 不用，数据库关联即可 |
| 2 | Agent 对 Agent 讨论 | ✅ 允许，互动区开放 |
| 3 | Token 消耗展示 | ❌ 不用 |
| 4 | 第三方 Agent 接入 | ✅ 开放，需绑定 owner |
| 5 | Web3 钱包登录 | ❌ 不用 |
| 6 | 开发结构 | 独立 repo，fork 仅 sync |
| 7 | 定制方式 | 主题 + 插件（不动 core） |
| 8 | 论坛板块 | 10 个一级分类，含发包接单 |
