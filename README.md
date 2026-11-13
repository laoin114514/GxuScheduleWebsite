# Schedule Update Server

西大课栈 App 的版本更新服务（Gin + GORM + 阿里云 OSS + MySQL）。

- `POST /api/v1/releases/upload` —— 发版时上传 APK，**每个应用一把独立上传 key**（`X-API-Key`）
- `GET /api/v1/apps/{appKey}/latest?versionCode=xxx` —— App / 官网检查最新版本（公开，限流）
- `/api/v1/admin/*` —— 管理面：创建 app、签发/轮换/吊销上传 key、启停 app、
  release 下架/恢复、审计日志、用量统计（`Authorization: Bearer $ADMIN_API_KEY`）
- `GET /healthz` —— 健康检查（Docker healthcheck 用）
- `GET /` —— 官网介绍页（前端构建产物，由本服务同源静态托管）

## 鉴权模型

| 凭据 | 请求头 | 用途 | 服务端存储 |
|---|---|---|---|
| 上传 key（每个 app 一把） | `X-API-Key: gxu_live_<keyId8>_<43字符>` | `POST /api/v1/releases/upload` | 只存 SHA-256；明文只在 admin 签发/轮换时返回一次 |
| 管理密钥（可多把） | `Authorization: Bearer $ADMIN_API_KEY`（也接受 `X-Admin-Key`） | `/api/v1/admin/*` 全部接口 | `.env` 的 `ADMIN_API_KEYS`（逗号分隔，支持轮换） |

- 上传 key 由服务端反查其绑定的 app，没有全局上传 key；表单 `appKey` 可省略，传了必须一致，否则 403 `APP_KEY_MISMATCH`。
- app 只能由管理面创建；上传者不能创建 app、不能改 app 名称/配置。
- 每个 app 同一时刻最多 1 把 `active` key；轮换时旧 key 进入 `retiring` 宽限期（默认 24h）后失效。
- 管理密钥绝不能用于上传；上传 key 也不能调用管理面。
- 设计与运维细节见 [`docs/per-app-upload-keys.md`](docs/per-app-upload-keys.md)。

## 部署（Docker Compose）

```bash
cp .env.example .env
# 编辑 .env：
#   ADMIN_API_KEYS=<openssl rand -hex 32>   # 必填
#   DB_*、OSS_*、CDN_BASE_URL 按实际填写
docker compose up -d --build
```

容器内 `DB_HOST` / `DB_PORT` / `SERVER_PORT` / `WEB_DIR` 由 `docker-compose.yml` 固定，
不需要写进 `.env`；限流、上传大小等使用 `internal/config` 的默认值。

`.env` 字段：

| 配置 | 含义 |
|---|---|
| `APP_ENV` | `production` 时 Gin 使用 release 模式 |
| `HOST_PORT` | 宿主暴露端口（容器内固定 8080） |
| `ADMIN_API_KEYS` | 管理面密钥，必填；逗号分隔多把实现无停机轮换 |
| `DB_NAME` / `DB_USER` / `DB_PASSWORD` | MySQL 库名/账号/密码 |
| `OSS_ENDPOINT` / `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` / `OSS_BUCKET` | 阿里云 OSS |
| `CDN_BASE_URL` | 非空时下载地址 = CDN + 对象 key；留空则返回 OSS 预签名 URL |

服务启动时 GORM AutoMigrate 会自动建表/补列（`apps`、`app_upload_keys`、`releases`、`audit_logs`），
生产首次部署建议先备份数据库；手工升级参考 [`migrations/001_per_app_keys.sql`](migrations/001_per_app_keys.sql)。

**改用外部数据库**（如阿里云 RDS）：在 `.env` 里设 `COMPOSE_DB_HOST=<RDS 地址>`，只起 app：

```bash
docker compose up -d --no-deps app
```

## 上传接口（Workflow / 脚本调用）

```bash
curl -X POST https://your-domain/api/v1/releases/upload \
  -H "X-API-Key: ${SCHEDULE_UPLOAD_KEY}" \
  -F "appKey=schedule" \
  -F "versionCode=107060" \
  -F "versionName=1.7.6" \
  -F "changelog=$(cat changelog.txt)" \
  -F "file=@app/build/outputs/apk/release/GxuScheduleAPP-release-1.7.6-universal.apk"
```

表单字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| appKey | 否 | 可省略；传了必须与该 key 绑定的 app 一致 |
| versionCode | 是 | 正整数，范围 `1..2100000000`，与 App 上报口径一致（见下方约定） |
| versionName | 是 | 如 `1.7.6` |
| file | 是 | APK 文件，文件名限 `[A-Za-z0-9._-]` 且扩展名 .apk/.ipa/.aab |
| changelog | 否 | 更新日志（release notes） |
| forced | 否 | 强制更新标记，默认 false |

行为约定：

- **不自动建 app**：key 解析不到 app 直接失败；新增渠道走管理面。
- **幂等**：同 `(appKey, versionCode)` 重复上传时 SHA256 一致返回 200；内容不同返回 409 `VERSION_CONFLICT`。
- **单调性**：默认拒绝低于当前最新 versionCode 的包（409 `DOWNGRADE_REJECTED`）；app 级 `allowDowngrade` 或全局
  `ALLOW_DOWNGRADE=true` 可放行。
- **配额/限流**：每 app 每日新增 release 上限（默认 20，幂等重传不计数）、单 key 令牌桶限流，超限 429。
- 大小上限取 `min(app.maxUploadMB, MAX_UPLOAD_MB)`（app 的 0 表示沿用全局）；超限 413。
- 成功响应字段与旧版一致；失败响应新增稳定 `errorCode`（见下方错误码表）。

成功响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "releaseId": 1,
    "appKey": "schedule",
    "versionCode": 107060,
    "versionName": "1.7.6",
    "fileKey": "release/schedule/107060/GxuScheduleAPP-release-1.7.6-universal.apk",
    "downloadUrl": "https://cdn.your-domain.com/release/schedule/107060/....apk",
    "sha256": "...",
    "size": 38689642
  }
}
```

失败响应：`{"code":401,"errorCode":"INVALID_API_KEY","message":"无效的上传密钥"}`。

### versionCode 口径（重要约定）

`app/build.gradle` 的 ABI splits 会把 versionCode 乘 10：APK manifest 里为 `base*10 + abi 码`
（universal = base*10），AGP 生成的 `BuildConfig.VERSION_CODE` 也是 base*10。
App 上报的就是这个值，Workflow 上传也必须用同一个数：

- Workflow 上传 `VERSION_CODE * 10`（universal 口径，如 tag v1.7.6 → 10706×10 = 107060）
- App 侧上报 `BuildConfig.VERSION_CODE`，与上面同值

## 管理面 API（`/api/v1/admin/*`）

鉴权：`Authorization: Bearer $ADMIN_API_KEY`（也接受 `X-Admin-Key`）。管理密钥只放服务器/管理员本机，
绝不进 CI、前端或仓库。所有写操作落 `audit_logs`；管理面按 IP 限流。

| 方法 | 路径 | 作用 |
|---|---|---|
| POST | `/apps` | 创建 app；`issueKey` 默认 `true`，可同时签发首把 key（明文只返回一次） |
| GET | `/apps` | app 列表 |
| GET | `/apps/{appKey}` | app 详情 + key 列表 |
| PATCH | `/apps/{appKey}` | 改名 / 启停（`status`：1 启用、2 停用）/ 配额、降级开关等 |
| POST | `/apps/{appKey}/keys` | 签发新 key（明文只在这次响应里） |
| GET | `/apps/{appKey}/keys` | key 列表（只含 `keyId/last4/status/...`，无明文） |
| POST | `/keys/{keyId}/rotate` | 轮换：签发新 key，旧 key 变 `retiring` 宽限（`graceHours` 默认 24，上限 720） |
| POST | `/keys/{keyId}/revoke` | 吊销 key（已吊销幂等 200） |
| GET | `/apps/{appKey}/releases` | release 列表；`?includeYanked=1&limit=100` |
| DELETE | `/releases/{releaseId}` | 下架（yank）release，body 可带 `{"reason":"..."}`；已下架幂等 200 |
| POST | `/releases/{releaseId}/restore` | 恢复已下架 release |
| GET | `/audit-logs` | 审计日志；`?appKey=&action=&limit=50` |
| GET | `/apps/{appKey}/usage` | 用量统计（今日上传数、release 数、key 数、最新版本等） |

```bash
ADMIN_API_KEY=$(openssl rand -hex 32)   # 与 .env 的 ADMIN_API_KEYS 一致

# 创建 app 并同时签发首把 key（data.uploadKey.plaintext 只出现一次）
curl -sS -X POST "https://your-domain/api/v1/admin/apps" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" -H "Content-Type: application/json" \
  -d '{"appKey":"schedule-beta","name":"西大课栈 Beta","issueKey":true,"label":"local-beta"}'

# 给已存在的 app 签发 key
curl -sS -X POST "https://your-domain/api/v1/admin/apps/schedule/keys" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" -H "Content-Type: application/json" \
  -d '{"label":"gha-release"}'

# 轮换 / 吊销
curl -sS -X POST "https://your-domain/api/v1/admin/keys/ab12cd34/rotate" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" -H "Content-Type: application/json" \
  -d '{"graceHours":24,"label":"gha-release"}'
curl -sS -X POST "https://your-domain/api/v1/admin/keys/ab12cd34/revoke" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" -H "Content-Type: application/json" \
  -d '{"reason":"suspected leak"}'

# 下架 / 恢复（release 用 releaseId 定位）
curl -sS -X DELETE "https://your-domain/api/v1/admin/releases/123" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" -H "Content-Type: application/json" \
  -d '{"reason":"1.7.7 闪退，回滚"}'
curl -sS -X POST "https://your-domain/api/v1/admin/releases/123/restore" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}"
```

### yank 的语义

- yank 只让 latest 跳过该版本，不删除 release 与 OSS 对象，可随时 restore。
- **不会让已安装高版本的用户降级**（Android 不允许）；真回滚要发一个 versionCode 更高的包。
- release 记录保留 `status/upload_key_id/yanked_*` 与审计。

## 错误码

失败响应统一为 `{"code":<HTTP 状态码>,"errorCode":"<稳定错误码>","message":"中文提示"}`；成功 `code=0`。
CI/脚本应按 `errorCode` 分支处理。

| errorCode | HTTP | 含义 |
|---|---|---|
| `INVALID_API_KEY` | 401 | 上传 key 缺失/格式错误/不存在/已吊销/已过期 |
| `ADMIN_UNAUTHORIZED` | 401 | 管理密钥缺失或错误 |
| `APP_NOT_FOUND` | 上传 403 / 管理 404 | app 不存在 |
| `APP_DISABLED` | 403 | app 已被停用 |
| `APP_KEY_MISMATCH` | 403 | 表单 appKey 与 key 绑定的 app 不一致 |
| `INVALID_PARAM` | 400 | 参数不合法（versionCode、versionName、文件名等） |
| `FILE_TOO_LARGE` | 413 | 超过 app/全局单文件上限 |
| `VERSION_CONFLICT` | 409 | 同 versionCode 已存在且 SHA256 不同 |
| `DOWNGRADE_REJECTED` | 409 | versionCode 低于当前最新且未允许降级 |
| `UPLOAD_QUOTA_EXCEEDED` | 429 | 该 app 当日新增 release 次数达上限 |
| `RATE_LIMITED` | 429 | 超过上传/管理面限流 |
| `APP_EXISTS` | 409 | 创建 app 时 appKey 已存在 |
| `KEY_NOT_FOUND` | 404 | keyId 不存在 |
| `KEY_NOT_ACTIVE` | 400 | 对非 active key 执行了只能对 active 做的操作 |
| `ACTIVE_KEY_EXISTS` | 409 | 该 app 已有 active key（每 app 同时仅 1 把） |
| `RELEASE_NOT_FOUND` | 404 | releaseId 不存在 |
| `INTERNAL` | 500 | 服务端内部错误 |
| `OSS_ERROR` | 500 | 对象存储写入/读取失败 |

## 最新版查询接口（App / 官网调用）

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

- `hasUpdate = current < latest`，latest 取该 appKey 下 `status=live` 的最大 versionCode（yanked 跳过）。
- 未知 appKey 返回 200 空（官网在服务端还没建该渠道时也能正常工作）；
  disabled app 返回 404 `APP_NOT_FOUND`；appKey 非法返回 400 `INVALID_PARAM`。
- `publishedAt` 为发版时间（RFC3339，UTC）；`fileName` 为安装包原始文件名。
- 公开接口，按 IP 限流（`RATE_LIMIT_RPS` / `RATE_LIMIT_BURST`）。
- 官网也读这个接口（`versionCode=0` 取最新版）。
- 跨域：默认 `CORS_ALLOW_ORIGINS=*`；收紧时在 `.env` 填白名单。

## 接入 GitHub Actions

仓库 Secrets：

| Secret | 说明 |
|---|---|
| `UPDATE_SERVER_URL` | 后端地址，如 `https://your-domain` |
| `SCHEDULE_UPLOAD_KEY` | `schedule` 渠道专属上传 key（admin API 签发，形如 `gxu_live_...`） |

上传步骤核心命令：

```yaml
      - name: Upload universal APK to update server
        id: update_server
        continue-on-error: true
        env:
          UPDATE_SERVER_URL: ${{ secrets.UPDATE_SERVER_URL }}
          SCHEDULE_UPLOAD_KEY: ${{ secrets.SCHEDULE_UPLOAD_KEY }}
        run: |
          UNIVERSAL_APK=$(ls app/build/outputs/apk/release/*-universal.apk | head -1)
          curl -sS --fail-with-body --retry 3 --retry-all-errors \
            -X POST "${UPDATE_SERVER_URL}/api/v1/releases/upload" \
            -H "X-API-Key: ${SCHEDULE_UPLOAD_KEY}" \
            -F "appKey=schedule" \
            -F "versionCode=${{ steps.version.outputs.VERSION_CODE }}0" \
            -F "versionName=${{ steps.version.outputs.VERSION }}" \
            -F "changelog=$(cat /tmp/changelog.txt)" \
            -F "file=@${UNIVERSAL_APK}"
```

测试版（`schedule-beta` 渠道）不走 GitHub Actions，由本地 `scripts/beta_release.py` 上传，
key 填在 `.beta-release.env` 的 `UPDATE_API_KEY`，见 [`../docs/测试版构建与发布.md`](../docs/测试版构建与发布.md)。

## 本地开发

```bash
cp .env.example .env   # 填 ADMIN_API_KEYS、本地 MySQL/OSS 配置
go run .
```

本地验证管理面：

```bash
curl -sS -X POST "http://localhost:8080/api/v1/admin/apps" \
  -H "X-Admin-Key: $(grep '^ADMIN_API_KEYS=' .env | cut -d= -f2 | cut -d, -f1)" \
  -H "Content-Type: application/json" \
  -d '{"appKey":"schedule","name":"西大课栈","issueKey":true}'
```

用返回的明文 key 上传：

```bash
curl -X POST http://localhost:8080/api/v1/releases/upload \
  -H "X-API-Key: gxu_live_...." \
  -F "appKey=schedule" -F "versionCode=10001" -F "versionName=1.0.1" \
  -F "file=@test-universal.apk"
```

## 官网与静态托管（`frontend/`）

Vue 3 + Vite 单页站，构建产物由本服务同源静态托管，`docker compose up -d --build` 一条命令同时起官网与接口。

镜像三阶段构建：`node:22-alpine` 构建前端 → `golang:1.22-alpine` 编译后端 → `alpine` 运行镜像。

静态托管逻辑（`internal/web/static.go`）：

- 命中真实文件直接返回：`assets/*` 带 `immutable` 长缓存，其它静态资源 1 小时
- 其它 GET 回退 `index.html`（SPA）；入口 HTML 不缓存
- 未匹配到的 `/api/...` 返回 JSON 404
- `WEB_DIR` 为空或目录不存在时不注册，保持纯接口服务

构建参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `VITE_APP_KEY` | `schedule` | 官网读取的渠道 appKey（公开接口无需上传 key） |
| `VITE_API_BASE_URL` | 空 | 留空 = 同源 `/api`；官网单独部署到别的域名时填写 |

前端单独开发：

```bash
cd frontend && npm install && npm run dev   # http://localhost:5173
```
