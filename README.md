# Schedule Update Server

西大课栈 App 的版本更新服务（Gin + GORM + 阿里云 OSS + MySQL）。

- `POST /api/v1/releases/upload` —— GitHub Actions 发版时上传 APK（API Key 鉴权）
- `GET /api/v1/apps/{appKey}/latest?versionCode=xxx` —— App 检查最新版本（公开，限流）
- `GET /healthz` —— 健康检查（Docker healthcheck 用）
- `GET /` —— 官网介绍页（前端构建产物，由本服务同源静态托管，见下文）

## 部署（Docker Compose，国内源）

```bash
cp .env.example .env
# 编辑 .env：填 OSS 密钥、数据库密码、改 API Key、CDN_BASE_URL
docker compose up -d --build
```

镜像源：基础镜像走 `m.daocloud.io`，Go 依赖走 `goproxy.cn`，alpine apk 走阿里云镜像。

compose 会一并起 `mysql`（MySQL 8.0）和 `app` 两个服务，数据落在命名卷 `mysql-data`，
容器重建不丢。二者共用 `.env` 的数据库配置：

- `DB_NAME` 建成库；`DB_USER` 非 `root` 时由 `docker/mysql/init/01-create-app-user.sh`
  在首次启动时建同名账号并授权（`DB_USER=root` 则直接用镜像的 root）
- app 容器内固定连 `mysql:3306`（服务名即主机名），不用改 `.env` 的 `DB_HOST`
- 宿主想用 Navicat 等直连：`127.0.0.1:3306`，端口冲突可设 `MYSQL_HOST_PORT`
- 服务启动时自动建表（GORM AutoMigrate），无需手工建表

**改用外部实例**（如阿里云 RDS）：在 `.env` 里设 `COMPOSE_DB_HOST=<RDS 地址>`，
只起 app 服务，并预先建库建用户：

```bash
docker compose up -d --no-deps app
```

```sql
CREATE DATABASE schedule CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'schedule'@'%' IDENTIFIED BY 'change-me';
GRANT ALL PRIVILEGES ON schedule.* TO 'schedule'@'%';
```

- RDS 白名单需放行服务器 IP；`DB_HOST`/`COMPOSE_DB_HOST` 填 RDS 内网/公网地址
- 若本地已有 MySQL 容器，Docker Desktop 下 `DB_HOST` 填 `host.docker.internal`

数据库只存 OSS 对象 key（`release/{appKey}/{versionCode}/{file}`），
下载地址由响应时生成：`CDN_BASE_URL` 非空时拼 base URL；留空时返回 OSS 预签名 URL（桶私有场景）。

## 上传接口（Workflow 调用）

```bash
curl -X POST https://your-domain/api/v1/releases/upload \
  -H "X-API-Key: ${UPLOAD_API_KEY}" \
  -F "appKey=schedule" \
  -F "versionCode=107060" \
  -F "versionName=1.7.6" \
  -F "name=西大课栈" \
  -F "changelog=$(cat changelog.txt)" \
  -F "file=@app/build/outputs/apk/release/GxuScheduleAPP-release-1.7.6-universal.apk"
```

表单字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| appKey | 是 | 应用 key，`[a-z0-9_-]`，2-64 位 |
| versionCode | 是 | 正整数，与 App 上报口径一致（见下方约定） |
| versionName | 是 | 如 `1.7.6` |
| file | 是 | APK 文件，文件名限 `[A-Za-z0-9._-]` 且扩展名 .apk/.ipa/.aab |
| name | 否 | 应用显示名，首次上传注册 app 信息 |
| changelog | 否 | 更新日志（release notes） |
| forced | 否 | 强制更新标记，默认 false |

行为约定：

- **幂等**：同 `(appKey, versionCode)` 重复上传时，SHA256 一致视为成功（200）并返回已有记录；内容不同返回 409。
- **单调性**：默认拒绝发布低于当前最新 versionCode 的包；确需回滚时在 `.env` 设 `ALLOW_DOWNGRADE=true`。
- 大小上限 `MAX_UPLOAD_MB`（默认 512MB）。

响应示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "releaseId": 1,
    "appKey": "schedule",
    "versionCode": 10706,
    "versionName": "1.7.6",
    "fileKey": "release/schedule/10706/GxuScheduleAPP-release-1.7.6-universal.apk",
    "downloadUrl": "https://cdn.your-domain.com/release/schedule/10706/....apk",
    "sha256": "...",
    "size": 38689642
  }
}
```

### versionCode 口径（重要约定，已在模拟器实测）

`app/build.gradle` 的 ABI splits 会把 versionCode 乘 10：APK manifest 里为 `base*10 + abi 码`
（universal = base*10），**且实测 AGP 8.7.3 生成的 `BuildConfig.VERSION_CODE` 也是 base*10**
（各 ABI 相同）。App 上报的就是 BuildConfig 的值，因此 Workflow 上传也必须用同一个数，
否则 App 的 versionCode（如 107060）永远大于库里存的 10706，"检查更新"永远判成已是最新：

- Workflow 上传 `VERSION_CODE * 10`（universal 口径，如 tag v1.7.6 → 10706×10 = 107060）
- App 侧上报 `BuildConfig.VERSION_CODE`，与上面同值（107060）

## 最新版查询接口（App 调用）

```bash
curl "https://your-domain/api/v1/apps/schedule/latest?versionCode=10706"
```

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "appKey": "schedule",
    "currentVersionCode": 10706,
    "hasUpdate": true,
    "latestVersionCode": 10716,
    "latestVersionName": "1.7.7",
    "changelog": "...",
    "downloadUrl": "https://cdn.your-domain.com/release/schedule/10716/....apk",
    "sha256": "...",
    "fileSize": 38689642,
    "forced": false,
    "publishedAt": "2026-08-31T15:00:47Z",
    "fileName": "GxuScheduleAPP-release-1.7.7-universal.apk"
  }
}
```

- `hasUpdate = current < latest`，`latest` 取该 appKey 下最大 versionCode。
- 无任何发布记录时返回 `hasUpdate=false`，字段为空。
- `publishedAt` 是发版时间（RFC3339，UTC，取自 `releases.created_at`），官网用它显示「更新日期」；
  `fileName` 是安装包原始文件名。
- **官网也读这个接口**（`versionCode=0` 永远拿最新版）：见 [`frontend/README.md`](frontend/README.md)。
- 该接口公开，按 IP 限流（`RATE_LIMIT_RPS` / `RATE_LIMIT_BURST`）。
- 跨域：官网若与更新服务不同域，需要 CORS。默认 `CORS_ALLOW_ORIGINS=*` 放行任意来源；
  想收紧就在 `.env` 里填白名单（逗号分隔，填官网域名），预检请求返回 204。

## 接入 GitHub Actions

在 `release.yml` 的"Extract release notes from tag"步骤之后、发布 GitHub Release 之前插入
（软失败：后端临时故障不阻断官方 Release）：

```yaml
      - name: Upload universal APK to update server（软失败）
        env:
          UPDATE_SERVER_URL: ${{ secrets.UPDATE_SERVER_URL }}
          UPDATE_SERVER_API_KEY: ${{ secrets.UPDATE_SERVER_API_KEY }}
        run: |
          UNIVERSAL_APK=$(ls app/build/outputs/apk/release/*-universal.apk 2>/dev/null | head -1) || true
          if [ -z "$UNIVERSAL_APK" ]; then
            echo "universal apk not found, skip"
            exit 0
          fi
          # versionCode 口径见下方说明：必须是 tag 派生值的 10 倍（universal 口径）
          UNIVERSAL_VERSION_CODE=$(( ${{ steps.version.outputs.VERSION_CODE }} * 10 ))
          curl -sS -X POST "${UPDATE_SERVER_URL}/api/v1/releases/upload" \
            -H "X-API-Key: ${UPDATE_SERVER_API_KEY}" \
            -F "appKey=schedule" \
            -F "versionCode=${UNIVERSAL_VERSION_CODE}" \
            -F "versionName=${{ steps.version.outputs.VERSION }}" \
            -F "name=西大课栈" \
            -F "changelog=${{ steps.release_notes.outputs.notes }}" \
            -F "file=@${UNIVERSAL_APK}" \
            || echo "::warning::上传更新服务器失败(忽略，GitHub Release 已正常发布)"
```

需要在仓库 Settings → Secrets 里配置：

| Secret | 说明 |
|---|---|
| `UPDATE_SERVER_URL` | 后端地址，如 `https://update.example.com` |
| `UPDATE_SERVER_API_KEY` | 与服务器 `.env` 的 `UPLOAD_API_KEY` 一致 |

## 本地开发

```bash
copy .env.example .env   # 填好本地 MySQL/OSS 配置
go run .
```

只想起数据库时，单独跑 compose 里的 mysql 服务即可，它的 3306 已映射到宿主，
`.env` 保持 `DB_HOST=127.0.0.1` 就能让本机 `go run .` 连上：

```bash
docker compose up -d mysql
```

用本机安装的 MySQL 也可以，建好 `DB_NAME` 对应的库后 `go run .` 即可。

## 官网与一键启动（`frontend/` + 静态托管）

官网是 `website/frontend` 下的 Vue 3 + Vite + Tailwind 单页站，构建产物由本服务**同源静态托管**，
所以一条 compose 命令就能同时起接口和官网，也不用处理跨域：

```bash
docker compose up -d --build

# 官网      http://<host>:<HOST_PORT>/
# 更新接口  http://<host>:<HOST_PORT>/api/v1/apps/schedule/latest?versionCode=0
# 健康检查  http://<host>:<HOST_PORT>/healthz
```

镜像由 [`Dockerfile`](Dockerfile) 三阶段构建：

1. `node:22-alpine` → `npm ci && npm run build` 产出 `dist`
2. `golang:1.22-alpine` → 编译 `schedule-server`
3. `alpine` → 二进制 + 前端产物（放到 `/app/web`，启动时 `WEB_DIR=/app/web`）

静态托管逻辑在 [`internal/web/static.go`](internal/web/static.go)：

- 命中真实文件直接返回：`assets/*` 带 `immutable` 长缓存，其它静态资源 1 小时
- 其它 GET 回退 `index.html`（SPA）；入口 HTML 是 `no-cache`，发版后刷新即生效
- 未匹配到的 `/api/...` 返回 JSON 404，不会把 HTML 喂给前端
- `WEB_DIR` 为空或目录不存在时不注册，保持纯接口服务（只部署 App 后端时行为不变）

### 构建参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `VITE_APP_KEY` | `schedule` | 与上传 workflow 的 `appKey` 一致 |
| `VITE_API_BASE_URL` | 空 | 留空 = 请求同源 `/api`；只有官网单独部署到别的域名时才需要填 |

需要覆盖官网里的真实截图时，把 `VITE_SHOT_*` 写进 `frontend/.env.production` 再重新构建
（该文件会随构建上下文进镜像）。

### 不用 Docker 的等价跑法

```bash
cd website/frontend && npm install && npm run build
cd website && WEB_DIR=frontend/dist go run .
# 打开 http://localhost:8080/ 就是官网，接口在同一端口
```

前端单独开发（热更新 + `/api` 自动代理）：

```bash
cd website/frontend && npm run dev     # http://localhost:5173
```

官网细节（文案、课表数据、主题、截图占位、`npm run check:api` 体检）见
[`frontend/README.md`](frontend/README.md)。

