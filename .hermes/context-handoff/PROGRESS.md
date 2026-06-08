# Hermix — 进度状态

## ✅ 已完成

| 功能 | 验证方式 |
|------|---------|
| 深色主题（对齐 hermesagent.org.cn） | body `#041c1c` + 暖 cream `#ffe6cb` + Noto Serif SC 衬线体 |
| 10 个一级分类 + 25 子版块 | curl `/categories` → 200, Redis `category:{5..39}` |
| Agent 角标 + 帖子橙色边框 | HTML 含 `agent-badge` + `agent-post`，topic 页面渲染确认 |
| Agent 注册（Web 表单 + API） | `POST /agent/register` → uid + apiToken；`/register` 含勾选框 |
| Agent API：Token 自助/能力/Webhook/Skill | 全要素测试 31/31 通过 |
| Agent 档案页（owner 名显示） | `/user/testagent` → Owner：admin |
| Agent 审核队列（首帖入审核） | Agent 发帖 → `queued:true` |
| Agent 限频（3帖/分 + 10000字） | plugin hook `checkAgentLimits` |
| Agent 可见性筛选（真人/Agent 按钮） | `filter:category.build` + `?agentFilter=` |
| 元数据帖子（JSON metadata） | Redis `post:{pid}` 含 metadata JSON 字段 |
| Webhook（回复 + @mention 通知） | `action:topic.reply` + `detectAgentMention` |
| SSRF 防护（webhook URL 校验） | localhost/192.168/10.x 被拦截 → `bad-request` |
| 移动端适配（3 断点） | 991px/767px/399px @media 查询 |
| 错误页（404/403/500 深色主题） | curl → 200 + 页面含 Hermix 样式 |
| 文档页 /docs | curl → 200, HTML 含 8 段 API 文档 |
| 全要素测试脚本 | `bash scripts/test-all.sh` → 31/31 pass |

## 🔄 进行中

无。当前会话已完成全部计划步骤。

## 📋 下次从哪干

### 第一条指令

```
cd /home/ayin/hermix
fuser -k 4567/tcp               # ← 必须！杀死僵尸进程
dev/nodebb/nodebb dev 2>&1 &    # 或 run_background
sleep 5
bash scripts/test-all.sh        # 确认 31/31 通过
```

### 就绪条件
- Redis 运行中（localhost:6379, db 1, 密码 myredissecret）
- Node.js >= 22
- `dev/nodebb/` 已通过 `scripts/setup-dev.sh` 搭建

### 已知陷阱

1. **端口冲突**：每次重启前必须 `fuser -k 4567/tcp`，否则 cluster worker 孤儿占端口。
2. **`./nodebb build` 失败**：因为 `dev/nodebb/scss/overrides.scss` 不存在。已在 `theme/scss/overrides.scss` 提供，`theme.json` 的 `baseTheme` 会正确解析。
3. **`db.setObjectBulk` 不存在**：用 `db.setObjectField` 逐个设置或用 `db.setObject` 全量覆盖。
4. **FormatApiResponse 不认 409**：用 400 替代。
5. **新 Agent 发帖**：NodeBB 新用户有 10s 冷却 + 120s 间隔。测试时用 admin token 发帖或等冷却。

### 未解决的问题（P2/远期）

- N+1 查询优化（getAgentList 等循环中有单独 DB 调用）
- Webhook 无重试/无失败日志
- PRD P2：Release Bot、GitHub SSO、Hermes WebUI 互通
- 无障碍 ARIA 标签
- 管理员面板功能增强（当前只有统计数字）
- 移动端 puppeteer 不可用，响应式效果未视觉验证
