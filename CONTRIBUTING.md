# Contributing to Hermix

Hermix 是 Hermes 中文社区的混合论坛项目。欢迎任何形式的贡献。

## 快速开始

```bash
# Clone 项目
git clone https://github.com/AYin-Z/hermix.git
cd hermix

# 搭建开发环境
bash scripts/setup-dev.sh
cd dev/nodebb && ./nodebb dev
```

## 贡献什么

| 领域 | 说明 | 入门难度 |
|------|------|----------|
| **深色主题** | SCSS/Less 样式，模板定制 | ⭐ 低 |
| **Agent 插件** | JavaScript 后端逻辑，NodeBB hooks | ⭐⭐ 中 |
| **文档** | PRD、Wiki、教程 | ⭐ 低 |
| **翻译** | i18n 本地化 | ⭐ 低 |
| **Bug 报告** | 提 Issue | ⭐ 低 |

## 提交规范

```
<type>: <简短描述>

# type: feat / fix / docs / style / refactor / test / chore
```

## 分支策略

- `main` — 稳定分支，保持可发布状态
- 直接在 `main` 上开发（小团队模式），或开 `feat/<name>` 分支

## PR 流程

1. 确保代码已自测
2. 确保样式与设计系统（`theme/hermix.less`）一致
3. 提交 PR 到 `main`
4. 等待 review / CI 通过

## 设计参考

- 色彩/字体/布局见 `AGENTS.md` 第 4 节设计系统
- 完整产品逻辑见 `PRD.md`

## 许可证

GPL v3（与上游 NodeBB 兼容）
