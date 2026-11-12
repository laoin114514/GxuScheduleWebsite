# Schedule Update Server

西大课栈 App 的版本更新服务（Gin + GORM + 阿里云 OSS + MySQL）。

- `POST /api/v1/releases/upload` —— 发版时上传 APK，**每应用独立上传 key** 鉴权（`X-API-Key`）
- `GET /api/v1/apps/{appKey}/latest?versionCode=xxx` —— App / 官网检查最新版本（公开，限流）
- `/api/v1/admin/*` —— 管理面：创建 app、签发/轮换/吊销上传 key、启用/停用 app、
  release 下架(yank)/恢复(restore)、审计日志、用量统计（`Authorization: Bearer $ADMIN_API_KEY`）
- `GET /healthz` —— 健康检查（Docker healthcheck 用；新增 `authMode` 字段显示当前上传鉴权模式）
- `GET /` —— 官网介绍页（前端构建产物，由本服务同源静态托管，见下文）

## 鉴权模型（先读这个）

| 凭据 | 请求头 | 用途 | 服务端存储 |
|---|---|---|---|
| 上传 key（每个 app 独立） | `X-API-Key: gxu_live_<keyId8>_<43字符>` | `POST /api/v1/releases/upload` | 只存 SHA-256；明文只在 admin 签发/轮换时返回一次 |
| 管理密钥（可多把） | `Authorization: Bearer $ADMIN_API_KEY`（也接受 `X-Admin-Key`） | `/api/v1/admin/*` 全部接口 | `.env` 的 `ADMIN_API_KEYS`（逗号分隔，支持轮换） |

- 上传 key 由服务端反查其绑定的 app，**不再有全局上传 key**；表单 `appKey` 可省略，
  传了必须与 key 绑定的 app 一致，否则 403 `APP_KEY_MISMATCH`。
- **上传者不能创建 app、不能改 app 名称/配置**；app 只能由管理面创建（不再「首次上传自动建 app」）。
- 每个 app 同一时刻最多 1 把 `active` key；轮换时旧 key 进入 `retiring` 宽限期（默认 24h）后失效。
- 管理密钥绝不能用于上传；上传 key 也绝不能调用管理面。
- 设计与运维细节（表结构、key 生命周期、迁移/回退 runbook）见
  [`docs/per-app-upload-keys.md`](docs/per-app-upload-keys.md)。

## 部署（Docker Compose，国内源）

```bash
cp .env.example .env
# 编辑 .env：
#   ADMIN_API_KEYS=<openssl rand -hex 32>   # 必填，否则服务启动失败（管理面不允许裸奔）
#   UPLOAD_AUTH_MODE=scoped                 # 全新部署用 scoped；已有服务迁移期用 dual（见 runbook）
#   DB_PASSWORD、OSS_*、CDN_BASE_URL 等按需填写
docker compose up -d --build
```

镜像源：基础镜像走 `m.daocloud.io`，Go 依赖走 `goproxy.cn`，alpine apk 走阿里云镜像。

compose 会一并起 `mysql`（MySQL 8.0）和 `app` 两个服务，数据落在命名卷 `mysql-data`，
容器重建不丢。二者共用 `.env` 的数据库配置：

- `DB_NAME` 建成库；`DB_USER` 非 `root` 时由 `docker/mysql/init/01-create-app-user.sh`
  在首次启动时建同名账号并授权（`DB_USER=root` 则直接用镜像的 root）
- app 容器内固定连 `mysql:3306`（服务名即主机名），不用改 `.env` 的 `DB_HOST`
- 宿主想用 Navicat 等直连：`127.0.0.1:3306`，端口冲突可设 `MYSQL_HOST_PORT`
- 服务启动时自动建表（GORM AutoMigrate）。本次升级新增 `app_upload_keys`、`audit_logs`
  两张表，`apps` / `releases` 增加新列（全部带默认值，存量数据自动回填，无需手工迁移）。
  生产首次升级建议先备份，也可手工执行 [`migrations/001_per_app_keys.sql`](migrations/001_per_app_keys.sql)

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

上传鉴权相关配置（其余见 `.env.example` 注释）：

| 配置 | 默认 | 含义 |
|---|---|---|
| `ADMIN_API_KEYS` | 无（必填） | 管理面密钥，逗号分隔可配多把；`openssl rand -hex 32` 生成 |
| `UPLOAD_AUTH_MODE` | `scoped` | `scoped` 只认 app 专属 key；`dual` 迁移期兼容旧全局 key；`legacy` 仅回滚用 |
| `UPLOAD_API_KEY` | 空 | 旧版全局上传 key，仅 `dual`/`legacy` 模式使用（迁移/回退期保留） |
| `UPLOAD_LEGACY_APPKEYS` | 空（不限制） | `dual` 模式旧全局 key 允许上传的渠道白名单，如 `schedule,schedule-beta` |
| `DEFAULT_DAILY_UPLOAD_LIMIT` | `20` | 新建 app 的每日上传次数上限（按自然日，幂等重传不计数） |
| `UPLOAD_RATE_RPS` / `UPLOAD_RATE_BURST` | `0.2` / `3` | 单把上传 key 的令牌桶限流（默认约 12 次/分钟） |
| `ADMIN_RATE_RPS` / `ADMIN_RATE_BURST` | `0.1` / `5` | 管理面按 IP 限流 |

## 上传接口（Workflow / 脚本调用）

每个 app 用自己的一把 key；下面示例中的 key 请替换成该渠道的真实 key（只存在于 GitHub Secrets、
`.beta-release.env` 或密码管理器里，**不要提交进仓库**）：

```bash
curl -X POST https://your-domain/api/v1/releases/upload \
  -H "X-API-Key: ${SCHEDULE_UPLOAD_KEY}" \
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
| appKey | 否 | 可省略；传了必须与该 key 绑定的 app 一致，否则 403 `APP_KEY_MISMATCH` |
| versionCode | 是 | 正整数，范围 `1..2100000000`，与 App 上报口径一致（见下方约定） |
| versionName | 是 | 如 `1.7.6` |
| file | 是 | APK 文件，文件名限 `[A-Za-z0-9._-]` 且扩展名 .apk/.ipa/.aab |
| name | 否 | **已废弃**：app 由管理面创建；对已存在的 app 该字段被忽略，上传者无权改名 |
| changelog | 否 | 更新日志（release notes） |
| forced | 否 | 强制更新标记，默认 false |

行为约定：

- **不自动建 app**：key 解析不到 app 直接失败；要新增渠道请走管理面 admin API。
- **幂等**：同 `(appKey, versionCode)` 重复上传时，SHA256 一致视为成功（200）并返回已有记录；
  内容不同返回 409 `VERSION_CONFLICT`。若该版本此前被下架（yanked），同一 APK 重传会把它恢复为 live。
- **单调性**：默认拒绝发布低于该渠道当前最新 versionCode 的包（409 `DOWNGRADE_REJECTED`）；
  app 级 `allowDowngrade` 或全局 `ALLOW_DOWNGRADE=true` 任一开启即可放行。
- **配额/限流**：按 app 的 `dailyUploadLimit`（默认 20/自然日，幂等重传不计数）与
  单 key 令牌桶（`UPLOAD_RATE_RPS/BURST`）限制，超限分别返回 429 `UPLOAD_QUOTA_EXCEEDED` / `RATE_LIMITED`。
- 大小上限取 `min(app.maxUploadMB, MAX_UPLOAD_MB)`（app 的 0 表示沿用全局）；超限 413 `FILE_TOO_LARGE`。
- 鉴权模式：`scoped` 只认 app key；`dual` 先认 app key、失败再回退旧全局 key（受白名单限制）；
  `legacy` 只认旧全局 key（仅回滚用）。

成功响应（字段与旧版一致）：

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

失败响应新增稳定错误码 `errorCode`（`code` 仍是 HTTP 状态码，便于老客户端兼容）：

```json
{ "code": 401, "errorCode": "INVALID_API_KEY", "message": "无效的上传密钥" }
```

### versionCode 口径（重要约定，已在模拟器实测）

`app/build.gradle` 的 ABI splits 会把 versionCode 乘 10：APK manifest 里为 `base*10 + abi 码`
（universal = base*10），**且实测 AGP 8.7.3 生成的 `BuildConfig.VERSION_CODE` 也是 base*10**
（各 ABI 相同）。App 上报的就是 BuildConfig 的值，因此 Workflow 上传也必须用同一个数，
否则 App 的 versionCode（如 107060）永远大于库里存的 10706，"检查更新"永远判成已是最新：

- Workflow 上传 `VERSION_CODE * 10`（universal 口径，如 tag v1.7.6 → 10706×10 = 107060）
- App 侧上报 `BuildConfig.VERSION_CODE`，与上面同值（107060）

## 管理面 API（`/api/v1/admin/*`）

鉴权：`Authorization: Bearer $ADMIN_API_KEY`（也接受 `X-Admin-Key: $ADMIN_API_KEY`）。
管理密钥只放管理员本机/服务器，**绝不进 CI、前端或仓库**。所有写操作会落 `audit_logs`。
管理面按 IP 限流（`ADMIN_RATE_RPS/BURST`），超限 429 `RATE_LIMITED`。
下面示例里的 `$ADMIN_API_KEY` shell 变量填 `.env` 中 `ADMIN_API_KEYS` 的任意一把即可。

| 方法 | 路径 | 作用 |
|---|---|---|
| POST | `/apps` | 创建 app；`issueKey` 默认 `true`，可同时签发首把 key（明文只返回一次） |
| GET | `/apps` | app 列表 |
| GET | `/apps/{appKey}` | app 详情 + 该 app 的 key 列表 |
| PATCH | `/apps/{appKey}` | 改名 / 启停（`status`：1 启用、2 停用）/ 改配额、降级开关等 |
| POST | `/apps/{appKey}/keys` | 签发新 key（明文只在这次响应里） |
| GET | `/apps/{appKey}/keys` | key 列表（只含 `keyId/last4/status/...`，无明文） |
| POST | `/keys/{keyId}/rotate` | 轮换：签发新 key，旧 key 变 `retiring` 宽限（`graceHours` 默认 24，上限 720） |
| POST | `/keys/{keyId}/revoke` | 吊销 key（已吊销幂等 200） |
| GET | `/apps/{appKey}/releases` | release 列表；`?includeYanked=1&limit=100` |
| DELETE | `/releases/{releaseId}` | 下架（yank）release，body 可带 `{"reason":"..."}`；已下架幂等 200 |
| POST | `/releases/{releaseId}/restore` | 恢复已下架 release |
| GET | `/audit-logs` | 审计日志；`?appKey=&action=&limit=50` |
| GET | `/apps/{appKey}/usage` | 用量统计（今日上传数、release 数、key 数、最新版本等） |

> release 用 **releaseId** 定位（不是 versionCode），列表接口会返回它；
> 也可以用 `GET /apps/{appKey}/releases` 先查。

### curl 示例

创建 app 并同时签发首把 key（已有 app 会返回 409 `APP_EXISTS`）：

```bash
curl -sS -X POST "https://your-domain/api/v1/admin/apps" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"appKey":"schedule-beta","name":"西大课栈 Beta","issueKey":true,"label":"local-beta"}'
# → data.app 是 app 信息；data.uploadKey.plaintext 是唯一一次可见的明文 key
```

给已存在的 app 签发新 key（`keyId` 与 `last4` 可用于定位，明文只在 `data.plaintext`）：

```bash
curl -sS -X POST "https://your-domain/api/v1/admin/apps/schedule/keys" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"label":"gha-release"}'
```

轮换 key（旧 key 进入 `retiring` 宽限期，到期自动失效；响应同时返回新明文和旧 key 信息）：

```bash
curl -sS -X POST "https://your-domain/api/v1/admin/keys/ab12cd34/rotate" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"graceHours":24,"label":"gha-release"}'
# → data.uploadKey.plaintext 填到 CI secret / .beta-release.env；data.retireKey 是旧 key
```

吊销 key（怀疑泄露时立即执行；轮换也可用吊销代替等待宽限）：

```bash
curl -sS -X POST "https://your-domain/api/v1/admin/keys/ab12cd34/revoke" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"reason":"suspected leak"}'
```

下架一个发错的 release（只是从 latest 隐藏；**已安装用户不会降级**，见下）：

```bash
# 先查 releaseId
curl -sS "https://your-domain/api/v1/admin/apps/schedule/releases?limit=10" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}"
# 下架
curl -sS -X DELETE "https://your-domain/api/v1/admin/releases/123" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"reason":"1.7.7 闪退，回滚"}'
# 恢复
curl -sS -X POST "https://your-domain/api/v1/admin/releases/123/restore" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}"
```

审计日志 / 用量统计：

```bash
curl -sS "https://your-domain/api/v1/admin/audit-logs?appKey=schedule&limit=50" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}"
curl -sS "https://your-domain/api/v1/admin/apps/schedule/usage" \
  -H "Authorization: Bearer ${ADMIN_API_KEY}"
```

### yank 的语义与陷阱（务必写进值班手册）

- **yank 不会让已安装高版本的用户降级**：Android 不允许 `versionCode` 降低，`latest` 跳过该版本后，
  已装 1.7.7 的用户查到的 latest 可能是 1.7.6，App 会判定「无更新」而不是降级。
  真正回滚的正确做法是**发一个 versionCode 更高的包**（内容为旧版或修复版），而不是只 yank。
- yank 只影响 latest 查询，不删除 release 记录与 OSS 对象，可随时 `restore`，也保留审计。
- **旧镜像不认识 release 状态**：回退到旧镜像前不要 yank；否则旧镜像会把已下架版本当成最新返回。
- 已上架版本被 yank 后，同版本同 SHA 重传会自动复活（幂等语义的一部分）。

## 错误码

失败响应统一为 `{"code":<HTTP 状态码>,"errorCode":"<稳定错误码>","message":"中文提示"}`；
成功响应 `code=0`。CI/脚本应按 `errorCode` 分支处理（`code` 可能随网关/代理变化）。

| errorCode | HTTP | 含义 | 处理建议 |
|---|---|---|---|
| `INVALID_API_KEY` | 401 | 上传 key 缺失/格式错误/不存在/已吊销/已过期 | 重新签发或轮换该渠道 key，更新 CI secret |
| `ADMIN_UNAUTHORIZED` | 401 | 管理密钥缺失或错误 | 检查 `Authorization: Bearer` / `X-Admin-Key` 与 `.env` 的 `ADMIN_API_KEYS` |
| `APP_NOT_FOUND` | 上传 403 / 管理 404 | app 不存在（上传时 appKey 未知；管理时查询目标不存在） | 先创建 app；确认 appKey 拼写 |
| `APP_DISABLED` | 403 | app 已被停用 | `PATCH /apps/{appKey}` 置 `status=1` 重新启用 |
| `APP_KEY_MISMATCH` | 403 | 表单 `appKey` 与 key 绑定的 app 不一致 | 用对应渠道的 key，或省略 `appKey` |
| `LEGACY_KEY_APP_NOT_ALLOWED` | 403 | `dual` 模式下旧全局 key 不在 `UPLOAD_LEGACY_APPKEYS` 白名单 | 换专属 key，或把渠道加入白名单（迁移期临时） |
| `INVALID_PARAM` | 400 | 参数不合法（versionCode、versionName、文件名等） | 按提示修正请求 |
| `FILE_TOO_LARGE` | 413 | 超过 app/全局单文件上限 | 调整 `MAX_UPLOAD_MB` / app 的 `maxUploadMB` |
| `VERSION_CONFLICT` | 409 | 同 versionCode 已存在且 SHA256 不同 | 换更高 versionCode；同包重传是幂等的 |
| `DOWNGRADE_REJECTED` | 409 | versionCode 低于当前最新且未允许降级 | 提升 versionCode，或临时开启降级（不推荐） |
| `UPLOAD_QUOTA_EXCEEDED` | 429 | 该 app 当日新增 release 次数达上限 | 次日再试或调大 app 的 `dailyUploadLimit` |
| `RATE_LIMITED` | 429 | 超过上传/管理面限流 | 退避后重试 |
| `APP_EXISTS` | 409 | 创建 app 时 appKey 已存在 | 直接给已有 app 签发 key |
| `KEY_NOT_FOUND` | 404 | keyId 不存在 | 用 `GET /apps/{appKey}/keys` 查真实 keyId |
| `KEY_NOT_ACTIVE` | 400 | 对非 active key 执行了只能对 active 做的操作 | 查 key 列表确认状态 |
| `ACTIVE_KEY_EXISTS` | 409 | 该 app 已有 active key（每 app 同时仅 1 把） | 用 rotate 轮换，或先吊销旧 key |
| `RELEASE_NOT_FOUND` | 404 | releaseId 不存在 | 用 release 列表接口确认 releaseId |
| `INTERNAL` | 500 | 服务端内部错误（数据库等） | 看服务端日志与 `X-Request-ID` |
| `OSS_ERROR` | 500 | 对象存储写入/读取失败 | 检查 OSS 配置、网络与配额 |

> `SERVER_NOT_CONFIGURED`（503）只在 `ADMIN_API_KEYS` 为空时由中间件返回；config 已强制必填，
> 正常启动不会出现，故未列入上表。

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

- `hasUpdate = current < latest`，`latest` 取该 appKey 下 **status=live** 的最大 versionCode（yanked 自动跳过）。
- **未知 appKey 仍返回 200 空**（`hasUpdate=false`，字段为空）——官网在服务端还没建该渠道时也能正常工作。
- **已停用（disabled）的 app 返回 404 `APP_NOT_FOUND`**（官网会回退到内置信息）；
  `appKey` 格式非法（不满足 `^[a-z0-9][a-z0-9_-]{1,63}$`）返回 400 `INVALID_PARAM`。
- 无任何在架发布记录时返回 `hasUpdate=false`，字段为空。
- `publishedAt` 是发版时间（RFC3339，UTC，取自 `releases.created_at`），官网用它显示「更新日期」；
  `fileName` 是安装包原始文件名。
- **官网也读这个接口**（`versionCode=0` 永远拿最新版）：见 [`frontend/README.md`](frontend/README.md)。
- 该接口公开，按 IP 限流（`RATE_LIMIT_RPS` / `RATE_LIMIT_BURST`）。
- 跨域：官网若与更新服务不同域，需要 CORS。默认 `CORS_ALLOW_ORIGINS=*` 放行任意来源；
  想收紧就在 `.env` 里填白名单（逗号分隔，填官网域名），预检请求返回 204。

## 接入 GitHub Actions

在 `release.yml` 的"Extract release notes from tag"步骤之后、发布 GitHub Release 之前插入
（`continue-on-error` 软失败：后端临时故障不阻断官方 Release；GitHub Release 之后再用一步把上传失败标红）：

```yaml
      - name: Upload universal APK to update server
        id: update_server
        continue-on-error: true
        env:
          UPDATE_SERVER_URL: ${{ secrets.UPDATE_SERVER_URL }}
          # schedule 渠道专属 key（admin API 签发）；旧 secret 仅迁移兼容，scoped 后失效
          UPDATE_SERVER_API_KEY: ${{ secrets.SCHEDULE_UPLOAD_KEY || secrets.UPDATE_SERVER_API_KEY }}
          RELEASE_NOTES: ${{ steps.release_notes.outputs.notes }}
        run: |
          if [ -z "$UPDATE_SERVER_URL" ] || [ -z "$UPDATE_SERVER_API_KEY" ]; then
            echo "::warning::未配置 UPDATE_SERVER_URL / SCHEDULE_UPLOAD_KEY secrets，跳过上传更新服务器"
            exit 0
          fi
          UNIVERSAL_APK=$(ls app/build/outputs/apk/release/*-universal.apk 2>/dev/null | head -1 || true)
          if [ -z "$UNIVERSAL_APK" ]; then
            echo "::warning::未找到 universal APK，跳过上传更新服务器"
            exit 0
          fi
          printf '%s' "$RELEASE_NOTES" > /tmp/changelog.txt
          # versionCode 口径见下方说明：必须是 tag 派生值的 10 倍（universal 口径）
          UNIVERSAL_VERSION_CODE=$(( ${{ steps.version.outputs.VERSION_CODE }} * 10 ))
          curl -sS --fail-with-body \
            --retry 3 --retry-all-errors --retry-delay 3 --retry-max-time 600 \
            -X POST "${UPDATE_SERVER_URL}/api/v1/releases/upload" \
            -H "X-API-Key: ${UPDATE_SERVER_API_KEY}" \
            -F "appKey=schedule" \
            -F "versionCode=${UNIVERSAL_VERSION_CODE}" \
            -F "versionName=${{ steps.version.outputs.VERSION }}" \
            -F "name=西大课栈" \
            -F "changelog=</tmp/changelog.txt" \
            -F "file=@${UNIVERSAL_APK}"
```

再在 GitHub Release 步骤之后追加失败检查：

```yaml
      - name: Fail if update server upload failed
        if: steps.update_server.outcome == 'failure'
        run: |
          echo "::error::更新服务器上传失败；GitHub Release 已发布，请检查日志后重试或手动补传"
          exit 1
```

需要在仓库 Settings → Secrets 里配置：

| Secret | 说明 |
|---|---|
| `UPDATE_SERVER_URL` | 后端地址，如 `https://update.example.com` |
| `SCHEDULE_UPLOAD_KEY` | **必配**：`schedule` 渠道专属上传 key（admin API 签发，形如 `gxu_live_...`） |
| `UPDATE_SERVER_API_KEY` | 旧全局 key，仅迁移期回退用；切 `scoped` 后请删除 |

测试版（`schedule-beta` 渠道）不走 GitHub Actions，由本地 `scripts/beta_release.py` 上传，
key 填在 `.beta-release.env` 的 `UPDATE_API_KEY`，见
[`docs/测试版构建与发布.md`](../docs/测试版构建与发布.md) §2.2。

## 迁移 runbook（旧全局 key → per-app key）

**前置：备份数据库与 `.env`**（`mysqldump --single-transaction ... > backup.sql`，复制一份 `.env`）。

1. **部署前改配置**：在 `.env` 增加 `ADMIN_API_KEYS=<openssl rand -hex 32>`（可多把逗号分隔），
   设 `UPLOAD_AUTH_MODE=dual`，**保留** 原来的 `UPLOAD_API_KEY`；建议同时设
   `UPLOAD_LEGACY_APPKEYS=schedule,schedule-beta` 收紧旧 key 的可用渠道。
   ⚠️ 新镜像缺 `ADMIN_API_KEYS` 会**启动失败**（config 校验必填），务必先改 `.env` 再升级镜像。
2. **启动新镜像**：`docker compose up -d --build`。AutoMigrate 会补新表/新列；观察 `/healthz`
   的 `authMode` 是否为 `dual`。
3. **签发 key**：用 admin API 给两个渠道各签发一把（app 记录已由 AutoMigrate/存量数据保留）：
   `POST /api/v1/admin/apps/schedule/keys`、`POST /api/v1/admin/apps/schedule-beta/keys`
   （若某渠道不存在先 `POST /api/v1/admin/apps` 创建，`issueKey:true` 可直接拿到首把 key）。
4. **更新使用方**：GitHub 仓库 secret `SCHEDULE_UPLOAD_KEY` 填 `schedule` 渠道 key；
   本机 `.beta-release.env` 的 `UPDATE_API_KEY` 填 `schedule-beta` 渠道 key。
5. **观察 legacy 用量归零**：`GET /api/v1/admin/audit-logs?appKey=schedule&limit=50`
   （或查 `audit_logs` 表）确认不再出现 `actor_type=legacy` 的上传记录。
6. **切 scoped**：`.env` 改 `UPLOAD_AUTH_MODE=scoped` 并重启；确认流水线/本地发版都正常。
7. **收尾**：再观察一个发布周期，删除 `.env` 的 `UPLOAD_API_KEY`、`UPLOAD_LEGACY_APPKEYS`，
   删除旧 GitHub secret `UPDATE_SERVER_API_KEY`。

**回退**：`.env` 改回 `UPLOAD_AUTH_MODE=dual`（旧全局 key 仍在时）或 `legacy`，重启；必要时回滚旧镜像。
⚠️ 回退旧镜像前先确认**没有 yank 过的 release**（旧镜像不认识 release status，会把下架版本当最新）。
新表/新列对旧代码是向后兼容的（带默认值），无需回退数据库。

## 本地开发

```bash
copy .env.example .env   # 填好本地 MySQL/OSS 配置
# 本地也必须配一把 ADMIN_API_KEYS（随便一个 ≥16 位的随机串即可），否则启动失败
go run .
```

只想起数据库时，单独跑 compose 里的 mysql 服务即可，它的 3306 已映射到宿主，
`.env` 保持 `DB_HOST=127.0.0.1` 就能让本机 `go run .` 连上：

```bash
docker compose up -d mysql
```

本地验证上传/管理面（示例，key 用上一步 admin API 签发的明文）：

```bash
# 创建本地测试 app 并拿 key
curl -sS -X POST "http://localhost:8080/api/v1/admin/apps" \
  -H "X-Admin-Key: $(grep '^ADMIN_API_KEYS=' .env | cut -d= -f2 | cut -d, -f1)" \
  -H "Content-Type: application/json" \
  -d '{"appKey":"schedule","name":"西大课栈","issueKey":true}'

# 用拿到的明文 key 上传
curl -X POST http://localhost:8080/api/v1/releases/upload \
  -H "X-API-Key: gxu_live_...." \
  -F "appKey=schedule" -F "versionCode=10001" -F "versionName=1.0.1" \
  -F "file=@test-universal.apk"
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
| `VITE_APP_KEY` | `schedule` | 官网读取的渠道（公开 latest 接口不受上传 key 影响） |
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
