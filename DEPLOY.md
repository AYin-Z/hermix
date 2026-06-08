# Hermix Docker 部署

## 快速启动

```bash
# 1. 启动全部服务
docker compose -f docker-compose.prod.yml up -d

# 2. 首次运行需要初始化 NodeBB
docker compose -f docker-compose.prod.yml exec nodebb ./nodebb setup

# 3. 激活 Hermix 主题和插件
docker compose -f docker-compose.prod.yml exec nodebb ./nodebb activate nodebb-theme-hermix
docker compose -f docker-compose.prod.yml exec nodebb ./nodebb activate nodebb-plugin-hermix

# 4. 重建资源
docker compose -f docker-compose.prod.yml exec nodebb ./nodebb build

# 5. 访问 http://localhost:4567
```

## 服务说明

| 服务 | 端口 | 说明 |
|------|------|------|
| nodebb | 4567 | NodeBB v4 + Hermix 主题/插件 |
| redis | 6379 | 缓存 + session |
| postgres | 5432 | 主数据库 |

## 目录挂载

| 容器路径 | 宿主机路径 | 说明 |
|----------|-----------|------|
| `/usr/src/app/node_modules/nodebb-theme-hermix` | `./theme` | 主题源码 |
| `/usr/src/app/node_modules/nodebb-plugin-hermix` | `./plugin` | 插件源码 |
| `/usr/src/app/config.json` | `./config.prod.json` | 生产配置 |
| `/usr/src/app/public/uploads` | `./data/uploads` | 上传文件 |
| `/var/lib/postgresql/data` | `postgres_data` | 数据库持久化 |
| `/data` | `redis_data` | Redis 持久化 |

## 自定义配置

复制 `config.prod.example.json` 为 `config.prod.json`，修改：
- `url` — 你的域名
- `secret` — 随机字符串
- `database` — 选择 `redis` 或 `postgres`
- `redis` / `postgres` — 数据库连接信息

## 更新

```bash
# 拉取最新代码
git pull

# 重建 + 重启
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml exec nodebb ./nodebb build
```
