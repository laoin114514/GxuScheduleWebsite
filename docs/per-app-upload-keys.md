# 每应用上传密钥（per-app upload keys）：设计与运维手册

面向维护者与值班同学。本文说明「每个 app 一把上传 key + 管理面 admin API」的完整设计、
日常运维（签发/轮换/吊销、下架 release、审计、配额）以及从旧「单一全局 `UPLOAD_API_KEY`」
迁移/回退的 runbook。代码实现见：

- `internal/auth/key.go` —— key 生成、格式解析、SHA-256、常量时间比较
- `internal/auth/resolver.go` —— 明文 key → app + key 记录（DB 查询）
- `internal/middleware/auth.go` —— 管理面 `AdminAuth`（`Authorization: Bearer` / `X-Admin-Key`）
- `internal/handler/` —— 上传、latest、admin API 的实现
- `migrations/001_per_app_keys.sql` —— 生产显式升级参考（GORM AutoMigrate 会做等价变更）

---

## 1. 为什么要改

旧方案只有一个全局 `UPLOAD_API_KEY`，且上传时任意合法 `appKey` 会自动建 app：

- 任何拿到 key 的人都能给任意渠道发版、创建新渠道，key 泄露影响面 = 全部渠道；
- 无法按渠道撤销权限（只能换全局 key，同时影响正式版和测试版）；
- 上传者能改 app 名称、能无限创建 app，没有配额/审计；
- 发布出去的错误版本无法隐藏（只能发更高版本覆盖）。

新方案：

- **每个 app（渠道）一把 key**，服务端由 key 反查 app，key 泄露只影响单个渠道；
- app 只能由管理面创建，上传者不能改 app 名称/配置，不再自动建 app；
- key 可独立签发/轮换/吊销，每把 key 有状态、过期时间、最后使用时间；
- 上传有每日配额与单 key 限流，所有写操作有审计日志；
- release 支持下架（yank）/恢复（restore），便于发错版本时止血。

---

## 2. 数据模型

`apps`（渠道）—— 新增字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` / `app_key` / `name` / `platform` / `created_at` / `updated_at` | 原有 | `app_key` 全局唯一且**永久保留**：停用后也不允许别的应用复用 |
| `status` | `TINYINT`，默认 1 | 1 = active，2 = disabled（拒绝上传、latest 返回 404） |
| `max_upload_mb` | `BIGINT`，默认 0 | 0 = 沿用全局 `MAX_UPLOAD_MB`；>0 时取两者较小值 |
| `daily_upload_limit` | `BIGINT`，默认 20 | 每自然日允许新增的 release 数；幂等重传不计数 |
| `allow_downgrade` | `TINYINT(1)`，默认 0 | 是否允许上传低于当前最新的 versionCode（与全局 `ALLOW_DOWNGRADE` 取或） |

`app_upload_keys`（上传 key）—— 新表：

| 字段 | 说明 |
|---|---|
| `id` | 自增主键 |
| `app_id` | 所属 `apps.id` |
| `key_id` | **8 位 hex 公开标识**（明文 key 中的一段），用于查库、审计、界面展示，不是机密 |
| `key_hash` | `SHA256(完整明文 key)` 的 hex，唯一索引；**明文绝不落库** |
| `last4` | 明文末 4 位，仅用于列表展示 |
| `label` | 用途标签，如 `gha-release` / `local-beta`，方便区分 |
| `status` | 1 = active，2 = retiring（轮换宽限），3 = revoked（已吊销） |
| `expires_at` | 可空；为空表示不过期。retiring 宽限也用它兜底 |
| `last_used_at` | 最近使用时间（写请求触发，内部每分钟最多更新一次，避免每请求写库） |
| `created_at` / `revoked_at` / `revoke_reason` | 审计辅助 |

`releases`（发布记录）—— 新增字段：

| 字段 | 说明 |
|---|---|
| `status` | 1 = live（latest 可见），2 = yanked（已下架，latest 跳过） |
| `upload_key_id` | 本次上传使用的 `app_upload_keys.id`；用旧全局 key 上传时为 0 |
| `yanked_at` / `yanked_by` / `yank_reason` | 下架审计信息 |

`audit_logs`（操作审计）—— 新表：`request_id`、`actor_type`（`admin`/`app`/`legacy`/`anonymous`）、
`actor_key_id`、`action`、`app_key`、`target_type`、`target_id`、`ip`、`user_agent`、`detail`（JSON 文本）、
`created_at`。**审计只记录元信息，绝不落明文密钥或请求体中的文件内容。**

不变量：**每个 app 同一时刻最多一把 `status=1`（active）的 key**。数据库不建部分唯一索引
（轮换期需要 active + retiring 并存），由 admin 签发/轮换时在事务里锁 `apps` 行保证。

---

## 3. key 格式与存储

```
gxu_live_<keyId8>_<secret43>
│         │         └─ base64url(32 字节随机) = 256 bit 熵，43 字符，可能含 '-'/'_'
│         └─ 8 位小写 hex，公开标识（4 字节随机），用于按 key_id 查库
└─ 固定前缀；`live` 便于 gitleaks 等扫描器识别泄露
```

- 总长度固定 61 字符；解析时按**固定长度切分**而不是按 `_` split（secret 本身可能含 `_`）。
- 服务端计算 `SHA256(完整明文)`，存 `key_hash`。比较用常量时间（`crypto/subtle`），防时序侧信道。
- 为什么不用 bcrypt/Argon2：key 是 256 bit 高熵随机串，不需要抗字典的慢哈希；SHA-256 足以做到
  「库泄露后无法反推明文」，且按 `key_id` 查到记录后能快速做常量时间比对。
- 明文只在 **签发（issue）/轮换（rotate）** 的响应里返回一次，字段名 `plaintext`。
  之后任何列表/详情接口都只返回 `keyId`、`last4`、状态等元信息。**丢了只能轮换重发。**

---

## 4. 鉴权流程

### 4.1 上传（`POST /api/v1/releases/upload`）

```
请求头 X-API-Key: gxu_live_...
  │
  ├─ 1. ParseKey：格式校验，取出 key_id
  ├─ 2. 按 key_id 查 app_upload_keys
  │      └─ 查不到 → ErrKeyNotFound（对外 401 INVALID_API_KEY）
  ├─ 3. 常量时间比较 SHA256(请求 key) == key_hash
  │      └─ 不等 → ErrKeyNotFound（不区分「keyID 存在但 secret 错」，避免侧信道）
  ├─ 4. key 状态：active / retiring 允许；revoked → ErrKeyRevoked（401）
  │        expires_at 已过 → ErrKeyExpired（401）
  ├─ 5. 查所属 app：不存在 → ErrKeyNotFound（401）；
  │        app.status=disabled → ErrAppDisabled（403 APP_DISABLED）
  ├─ 6. 表单 appKey：app 专属 key 可省略；传了必须等于 key 绑定的 app.app_key，
  │        否则 403 APP_KEY_MISMATCH；旧全局 key（legacy/dual 回退）必须显式传 appKey，
  │        且受 UPLOAD_LEGACY_APPKEYS 白名单限制，未知 appKey 返回 403 APP_NOT_FOUND
  └─ 7. 后续校验（配额/限流/大小/versionCode/降级/幂等）→ 落库 + OSS，写审计
```

- **全程不自动建 app**：key 一定绑定到已存在的 app；上传者无法创建/修改 app。
- **管理密钥被显式拒绝**：拿 `ADMIN_API_KEYS` 里的值当上传 key 会返回 401 `INVALID_API_KEY`
  （服务端专门判断，避免管理密钥扩散到 CI/发布脚本）。
- 旧全局 key 只在 `dual`/`legacy` 模式参与鉴权：`dual` 先解析专属 key，失败再比旧 key，
  且旧 key 受 `UPLOAD_LEGACY_APPKEYS` 白名单限制（空 = 不限制；建议迁移期显式填）。
  用旧 key 上传的 release 记 `upload_key_id=0`，审计 `actor_type=legacy`。
- `scoped` 模式下旧全局 key 完全无效（返回 `INVALID_API_KEY`），但管理面 admin API 不受影响。
- 上传失败也写审计（谁在什么时候对哪个 app 用了哪把 key、失败原因），便于排查泄露与误配置。
- `last_used_at` 由 `TouchKeyLastUsed` 尽力更新，失败不影响上传。

### 4.2 管理面（`/api/v1/admin/*`）

- 请求头：`Authorization: Bearer $ADMIN_API_KEY`，也接受 `X-Admin-Key: $ADMIN_API_KEY`。
- 服务端 `ADMIN_API_KEYS` 支持逗号分隔多把，常量时间逐个比较，便于**无停机轮换管理密钥**。
- **`ADMIN_API_KEYS` 必填**：缺失时服务直接启动失败（`config.validate`），管理面绝不裸奔。
- 管理密钥与上传 key 完全隔离：上传 key 不能调管理面，管理密钥也不能用于上传。
- 管理面按 IP 限流（`ADMIN_RATE_RPS`/`ADMIN_RATE_BURST`），所有写操作写审计。

---

## 5. key 生命周期

```
            POST /apps/{appKey}/keys
                     │
                     ▼
   ┌──────────► active ────────── rotate ─────────► retiring ──(宽限期到/吊销)──► revoked
   │              │                                   │
   │              └──────── revoke ────────────────────┴──────────────────────────► revoked
   │
   └── 已吊销/过期后不可恢复，只能重新签发
```

| 操作 | 接口 | 语义 |
|---|---|---|
| 签发 | `POST /apps/{appKey}/keys` | 该 app 没有 active key 时创建；已有一把 active 时返回 `409 ACTIVE_KEY_EXISTS`（要换请用 rotate）。可带 `label`、可选 `expiresAt`（RFC3339） |
| 列表 | `GET /apps/{appKey}/keys` | 返回全部历史 key（含已吊销），无明文 |
| 轮换 | `POST /keys/{keyId}/rotate` | 同一事务里：旧 active → retiring 且 `expires_at = now + graceHours`（默认 24，上限 720）；签发新 active 并返回明文。响应含 `uploadKey`（新，带 `plaintext`）与 `retireKey`（旧） |
| 吊销 | `POST /keys/{keyId}/revoke` | 立即置 revoked；已吊销幂等 200。怀疑泄露时**先吊销再签发**，不要等宽限期 |
| 过期 | — | `expires_at` 到点后自动失效；retiring key 的宽限也靠这个字段兜底 |

轮换宽限期（grace）的意义：给 CI/本地脚本留出更新 key 的时间窗口。推荐流程：

1. `rotate` 拿到新明文，**立刻**更新 GitHub secret `SCHEDULE_UPLOAD_KEY` / `.beta-release.env`；
2. 观察 `audit_logs` 中旧 `key_id` 不再被使用（或等宽限期结束自动失效）；
3. 若怀疑 key 已泄露（比如误提交日志），不要 rotate 等宽限，直接 `revoke` 旧 key。

注意：

- `KEY_NOT_FOUND`（404）：keyId 不存在，先用 key 列表确认真实 keyId；
- `KEY_NOT_ACTIVE`（400）：对非 active key 做了只能对 active 做的操作（如再签发）；
- 每个 app 只有一把 active，**轮换不会让旧 key 立即失效**，所以轮换期间新旧 key 都能上传；
- 建议每个用途一把独立 key（`label` 区分 `gha-release`、`local-beta`、`manual-backfill`），
  这样单独吊销某个用途不会影响其它发版链路。

---

## 6. release 下架（yank）语义与陷阱

接口：`DELETE /api/v1/admin/releases/{releaseId}`（body 可带 `{"reason":"..."}`，已下架幂等 200）、
`POST /api/v1/admin/releases/{releaseId}/restore`。list 用
`GET /api/v1/admin/apps/{appKey}/releases?includeYanked=1&limit=100`，用返回的 **releaseId** 定位。

语义：

- yank 只是把 release 的 `status` 从 1（live）改成 2（yanked），latest 查询跳过它；
- **不删除** release 记录与 OSS 对象，可随时 restore，保留审计；
- 同版本同 SHA 重新上传会幂等命中并把该 release 复活为 live（相当于 restore）；
- 已安装更高版本的用户**不会降级**：Android 不允许更低 versionCode 覆盖安装。
  如果 latest 因此回退到更旧的版本，App 只会判定「无更新」，不会去装旧包。

**真正回滚发错版本的正确做法**：

1. 先 yank 掉有问题的版本，让还没升级的用户不再拿到它；
2. 立即**发一个 versionCode 更高**的包（内容为修复版，或退回旧版内容），让已经升级的用户也能拿到修复；
3. 高版本用户能否收到取决于新版 versionCode 是否高于他们当前安装的版本。

陷阱：

- **不能只 yank 就指望回滚**：已经升级的用户会停在问题版本。
- **旧镜像不认识 release status**：如果回退到旧镜像（旧代码按最大 versionCode 查 latest，不筛 status），
  yanked 版本会被当成最新返回。**回退旧镜像前先确认没有 yanked 记录**，或先 restore。
- `includeYanked=1` 才能看到已下架记录；默认列表只返回 live。
- yank 恢复的审计（`yanked_by`/`yank_reason`）会保留，便于事后复盘。

---

## 7. 上传配额与限流

| 机制 | 配置 | 默认 | 触发时 |
|---|---|---|---|
| 每 app 每日新增 release 数 | app 级 `dailyUploadLimit`（建 app 时未指定则用 `DEFAULT_DAILY_UPLOAD_LIMIT`） | 20/自然日 | 429 `UPLOAD_QUOTA_EXCEEDED` |
| 单把上传 key 的令牌桶 | `UPLOAD_RATE_RPS` / `UPLOAD_RATE_BURST` | 0.2 / 3（约 12 次/分钟） | 429 `RATE_LIMITED` |
| 旧全局 key（legacy/dual 回退）按客户端 IP | 同上 | 同上 | 429 `RATE_LIMITED` |
| 管理面按 IP | `ADMIN_RATE_RPS` / `ADMIN_RATE_BURST` | 0.1 / 5 | 429 `RATE_LIMITED` |
| 公开 latest 接口按 IP | `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` | 2 / 10 | 429 |

- 每日配额按自然日统计、只计「新增 release」；同版本同 SHA 的幂等重传不计数（重试安全）。
- 幂等重传即使超过当日配额也应成功（否则 CI 重试会被误杀）——实现以上传是否产生新记录为准。
- 单文件上限取 `min(app.maxUploadMB, MAX_UPLOAD_MB)`（`app.maxUploadMB=0` 表示沿用全局）。
- 配额/限流都按 key/app 维度隔离：beta 渠道的频繁重试不会挤占正式渠道。

调整方式：`PATCH /api/v1/admin/apps/{appKey}` 改 `dailyUploadLimit` / `maxUploadMB` / `allowDowngrade`，
无需重启服务；全局默认值改 `.env` 需重启。

---

## 8. latest 与 app 状态

`GET /api/v1/apps/{appKey}/latest?versionCode=xxx`（公开）：

| 场景 | 行为 |
|---|---|
| app 不存在（未知 appKey） | **200，`hasUpdate=false`、字段为空**（官网兼容：官网先于后端渠道上线也不报错） |
| app 已停用（status=2） | **404 `APP_NOT_FOUND`**（官网会回退到内置信息） |
| 正常 app | 只取 `status=live` 的最大 versionCode（yanked 自动跳过） |
| 无 live release | 200 空 |
| 请求过频 | 429 |

管理面停用/启用 app：`PATCH /api/v1/admin/apps/{appKey}` body `{"status":2}` / `{"status":1}`。
停用后：上传被拒（403 `APP_DISABLED`）、latest 404，但历史数据与 key 记录保留，启用后立即恢复。

`GET /healthz` 会返回当前 `authMode`（`scoped`/`dual`/`legacy`），迁移期用它确认配置是否生效。

---

## 9. 迁移 runbook（全局 key → per-app key）

**前置：备份数据库与 `.env`。**

```bash
mysqldump -u<user> -p<pass> --single-transaction <db> > backup_before_per_app_keys.sql
cp .env .env.bak.$(date +%Y%m%d%H%M)
```

### 9.1 切换到 dual（迁移期）

1. 编辑 `.env`：
   - 新增 `ADMIN_API_KEYS=<openssl rand -hex 32>`（多把用逗号分隔，可先只放一把）；
   - `UPLOAD_AUTH_MODE=dual`；
   - **保留** 原 `UPLOAD_API_KEY`（旧全局 key，dual 模式回退用）；
   - 建议设 `UPLOAD_LEGACY_APPKEYS=schedule,schedule-beta`，把旧 key 的可用渠道收紧到现有渠道。
   ⚠️ 新镜像缺 `ADMIN_API_KEYS` 会**启动失败**，务必先改 `.env` 再升级镜像。
2. `docker compose up -d --build` 启动新镜像；确认 `/healthz` 的 `authMode=dual`，
   数据库出现 `app_upload_keys` / `audit_logs` 表（AutoMigrate 自动完成）。
3. 用 admin API 给每个渠道签发专属 key（老 app 记录会被保留；不存在则先创建）：

   ```bash
   # 已存在 → 直接签发；首次创建时可以 issueKey:true 顺手拿首把 key
   curl -sS -X POST "${BASE}/api/v1/admin/apps/schedule/keys" \
     -H "Authorization: Bearer ${ADMIN_API_KEY}" \
     -H "Content-Type: application/json" -d '{"label":"gha-release"}'
   curl -sS -X POST "${BASE}/api/v1/admin/apps/schedule-beta/keys" \
     -H "Authorization: Bearer ${ADMIN_API_KEY}" \
     -H "Content-Type: application/json" -d '{"label":"local-beta"}'
   ```

4. 把 `schedule` 渠道的明文 key 写进 GitHub 仓库 secret `SCHEDULE_UPLOAD_KEY`；
   把 `schedule-beta` 渠道的明文填进本机 `.beta-release.env` 的 `UPDATE_API_KEY`。
5. 跑一次正式版/测试版发版验证新 key 可用；观察审计日志里 `actor_type=legacy` 的用量逐步归零：
   `GET /api/v1/admin/audit-logs?limit=50`（或直接查 `audit_logs` 表）。

### 9.2 切换到 scoped

1. 确认 legacy 上传已归零（含存量流水线都已换成新 secret）。
2. `.env` 改 `UPLOAD_AUTH_MODE=scoped`，重启。
3. 再发一次正式版与测试版验证；旧全局 key 此时应返回 `INVALID_API_KEY`。
4. 收尾：再观察一个发布周期后，删除 `.env` 的 `UPLOAD_API_KEY`、`UPLOAD_LEGACY_APPKEYS`，
   删除 GitHub 旧 secret `UPDATE_SERVER_API_KEY`，并修改原全局 key 的值（即使已不使用也作废）。

### 9.3 回退

- **快**：`.env` 改回 `UPLOAD_AUTH_MODE=dual`（旧 `UPLOAD_API_KEY` 还在时）或 `legacy`，重启即可；
  此时新旧 key 都可用（dual），或只认旧 key（legacy）。管理面 admin API 始终可用。
- **回滚镜像**：旧镜像不认识 `releases.status`，会忽略 yank；回退前确认**没有 yanked release**，
  必要时先 `restore`。同时旧镜像的 config 校验要求 `UPLOAD_API_KEY` 非空，回滚时别把它删了。
- **数据库**：新增表/列全部带默认值，对旧代码向后兼容，无需回滚数据库；
  但回滚后新代码写入的数据（如 disabled app、yanked release）会被旧代码忽略。

### 9.4 迁移检查清单

- [ ] 数据库与 `.env` 已备份
- [ ] `.env` 有 `ADMIN_API_KEYS`（≥16 字符，建议 `openssl rand -hex 32`）
- [ ] `UPLOAD_AUTH_MODE=dual` 且旧 `UPLOAD_API_KEY` 仍保留
- [ ] `healthz.authMode=dual`
- [ ] `schedule` / `schedule-beta` 各有一把 active key（`GET /apps/{appKey}/keys`）
- [ ] GitHub secret `SCHEDULE_UPLOAD_KEY` 已更新
- [ ] `.beta-release.env` 的 `UPDATE_API_KEY` 已更新
- [ ] 审计日志 legacy 上传归零
- [ ] 切 `scoped` 后两渠道各成功发一版
- [ ] 观察期后删除旧 key/旧 secret

---

## 10. 安全注意事项

1. **只存 hash**：服务端 `app_upload_keys.key_hash = SHA256(明文)`，数据库/备份泄露无法还原 key。
2. **明文只返回一次**：签发/轮换响应中的 `plaintext` 是唯一一次；不要把它复制进聊天、日志、工单，
   也不要提交到 git。拿到后立即写入目标位置（GitHub secret / `.beta-release.env`）。
3. **管理密钥不进 CI**：`ADMIN_API_KEYS` 只放管理员本机/服务器 `.env`；
   GitHub Actions 只配 `SCHEDULE_UPLOAD_KEY`（单渠道上传 key）。CI 日志、构建产物、前端代码里
   绝不能出现管理密钥。`.beta-release.env`、`.env` 均已被 `.gitignore` 忽略，仍应 `chmod 600`。
4. **每渠道/每用途一把 key**：正式渠道与测试渠道分开，CI 与本地脚本分开（`label` 标注用途）。
   一把 key 泄露时只吊销这一把，不影响其它链路。
5. **泄露处置**：立即 `POST /keys/{keyId}/revoke`（不要等宽限）→ 签发新 key 并更新所有使用方 →
   查 `audit_logs` 确认没有异常上传/下架操作 → 必要时把受影响 release 逐个核对。
6. **轮换**：建议正式渠道 CI key 每 90 天或人员变动时轮换；`rotate` 的 `graceHours` 只给更新配置留时间，
   不要长期留 retiring key。
7. **旧 key 必须下线**：迁移完成后删除 `UPLOAD_API_KEY`、`UPLOAD_LEGACY_APPKEYS` 与旧 secret，
   否则旧全局 key 仍是一个可被利用的大权限凭据（legacy 模式下可上传任意渠道）。
8. **截断展示**：日志/界面只展示 `keyId`（公开标识）与 `last4`；绝不打印完整明文或 `key_hash`。
9. **审计**：所有写操作落 `audit_logs`（含 `request_id` 与响应头 `X-Request-ID` 对应），
   排查时按 `request_id` 串联服务日志。审计 `detail` 只放元信息，不放明文与文件内容。
10. **停用与复用**：app 停用后 `app_key` 永久保留，不能给别的应用复用（防止 latest/更新渠道串味）；
    确需新渠道请新建 appKey。

---

## 11. 常见运维问答

**Q：审核/改版本号时能不能直接改数据库？**
A：不建议。改 `releases.status` 用 yank/restore 接口（有审计、幂等）；改 versionCode 会破坏
`(app_key, version_code)` 唯一约束与 App 端口径，回滚请发更高 versionCode。

**Q：上传返回 `APP_KEY_MISMATCH` 怎么办？**
A：key 属于 A 渠道但表单传了 B 的 appKey。CI/脚本应使用对应渠道的 key；`appKey` 可省略。

**Q：CI 上传返回 `INVALID_API_KEY`，但 key 看起来没变？**
A：确认 secret 是 `SCHEDULE_UPLOAD_KEY` 且已更新为「本渠道」的新 key；确认 key 未过期/未被吊销
（`GET /apps/schedule/keys` 看状态）；确认服务端已是 scoped 且旧 key 已停用。

**Q：某个版本被 yank 了，为什么已升级的用户还在用？**
A：yank 只影响未升级用户的 latest。已升级用户必须靠更高 versionCode 的新包覆盖（见 §6）。

**Q：误建了 app / 误发了 key 怎么办？**
A：app 无删除接口，可 `PATCH status=2` 停用；key 可 revoke。误发的 key 立即 revoke 即可
（同 app 已没有其它 active 时再签发新 key）。

**Q：管理面返回 `ADMIN_UNAUTHORIZED`？**
A：检查请求头格式（`Authorization: Bearer <key>` 或 `X-Admin-Key: <key>`）与 `.env` 的
`ADMIN_API_KEYS` 是否一致；多把时任一匹配即可；改 `.env` 后需重启容器。
