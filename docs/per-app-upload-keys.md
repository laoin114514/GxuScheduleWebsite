# 每应用上传密钥（per-app upload keys）：设计与运维手册

面向维护者与值班同学。本文说明「每个 app 一把上传 key + 管理面 admin API」的设计、
日常运维（签发/轮换/吊销、下架 release、审计、配额）与排障 。代码实现见：

- `internal/auth/key.go` —— key 生成、格式解析、SHA-256、常量时间比较
- `internal/auth/resolver.go` —— 明文 key → app + key 记录（DB 查询）
- `internal/middleware/auth.go` —— 管理面 `AdminAuth`
- `internal/handler/` —— 上传、latest、admin API
- `migrations/001_per_app_keys.sql` —— 建表/补列参考（GORM AutoMigrate 会自动执行等价变更）

## 1. 设计目标

- **每个 app（渠道）一把 key**：服务端由 key 反查 app，key 泄露只影响单个渠道；
- app 只能由管理面创建，上传者不能改 app 名称/配置、不能创建 app；
- key 可独立签发/轮换/吊销，有状态、过期时间、最后使用时间；
- 上传有每日配额与单 key 限流，所有写操作有审计日志；
- release 支持下架（yank）/恢复（restore），发错版本时可止血。

## 2. 数据模型

`apps`（渠道）：

| 字段 | 说明 |
|---|---|
| `id` / `app_key` / `name` / `platform` / `created_at` / `updated_at` | `app_key` 全局唯一且**永久保留**，停用后也不允许别的应用复用 |
| `status` | 1 = active，2 = disabled（拒绝上传、latest 返回 404） |
| `max_upload_mb` | 0 = 沿用全局 `MAX_UPLOAD_MB`；>0 时取两者较小值 |
| `daily_upload_limit` | 每自然日允许新增的 release 数；幂等重传不计数 |
| `allow_downgrade` | 是否允许上传低于当前最新的 versionCode（与全局 `ALLOW_DOWNGRADE` 取或） |

`app_upload_keys`（上传 key）：

| 字段 | 说明 |
|---|---|
| `app_id` | 所属 `apps.id` |
| `key_id` | 8 位 hex 公开标识（明文 key 中的一段），用于查库、审计、展示，不是机密 |
| `key_hash` | `SHA256(完整明文 key)` 的 hex，唯一索引；**明文绝不落库** |
| `last4` | 明文末 4 位，仅用于列表展示 |
| `label` | 用途标签，如 `gha-release` / `local-beta` |
| `status` | 1 = active，2 = retiring（轮换宽限），3 = revoked |
| `expires_at` | 可空；为空表示不过期。retiring 宽限也用它兜底 |
| `last_used_at` | 最近使用时间（每分钟最多更新一次） |
| `created_at` / `revoked_at` / `revoke_reason` | 审计辅助 |

`releases` 新增字段：`status`（1 live / 2 yanked）、`upload_key_id`（上传所用 key）、
`yanked_at` / `yanked_by` / `yank_reason`。

`audit_logs`：`request_id`、`actor_type`（`admin`/`app`/`anonymous`）、`actor_key_id`、
`action`、`app_key`、`target_type`、`target_id`、`ip`、`user_agent`、`detail`（JSON 文本）、`created_at`。
**审计只记录元信息，绝不落明文密钥。**

不变量：**每个 app 同一时刻最多一把 `status=1`（active）key**，由 admin 签发/轮换时在事务里锁 `apps` 行保证。

## 3. key 格式与存储

```
gxu_live_<keyId8>_<secret43>
│         │         └─ base64url(32 字节随机) = 256 bit 熵，43 字符，可能含 '-'/'_'
│         └─ 8 位小写 hex，公开标识（4 字节随机），用于按 key_id 查库
└─ 固定前缀；`live` 便于 gitleaks 等扫描器识别泄露
```

- 总长度固定 61 字符；解析按固定长度切分（secret 本身可能含 `_`）。
- 服务端存 `SHA256(完整明文)`，比较用常量时间（`crypto/subtle`）。
- 256 bit 高熵随机串不需要 bcrypt/Argon2；SHA-256 足够做到「库泄露后无法反推明文」。
- 明文只在 **签发（issue）/轮换（rotate）** 的响应里返回一次，字段名 `plaintext`；
  之后列表/详情只返回 `keyId`、`last4`、状态等。**丢了只能轮换重发。**

## 4. 鉴权流程

### 4.1 上传（`POST /api/v1/releases/upload`）

```
请求头 X-API-Key: gxu_live_...
  ├─ 1. ParseKey：格式校验，取出 key_id
  ├─ 2. 按 key_id 查 app_upload_keys；查不到 → 401 INVALID_API_KEY
  ├─ 3. 常量时间比较 SHA256(请求 key) == key_hash；不等 → 401
  ├─ 4. key 状态 active / retiring 允许；revoked → 401；expires_at 已过 → 401
  ├─ 5. app 不存在 → 401；app.status=disabled → 403 APP_DISABLED
  ├─ 6. 表单 appKey 可省略；传了必须等于 key 绑定的 app.app_key，否则 403 APP_KEY_MISMATCH
  └─ 7. 后续校验（配额/限流/大小/versionCode/降级/幂等）→ 落库 + OSS，写审计
```

- **全程不自动建 app**：key 一定绑定到已存在的 app；上传者无法创建/修改 app。
- **管理密钥被显式拒绝**：拿 `ADMIN_API_KEYS` 的值当上传 key 会返回 401 `INVALID_API_KEY`。
- 上传失败也写审计（谁在什么时候对哪个 app 用了哪把 key、失败原因）。
- `last_used_at` 由 `TouchKeyLastUsed` 尽力更新，失败不影响上传。

### 4.2 管理面（`/api/v1/admin/*`）

- 请求头：`Authorization: Bearer $ADMIN_API_KEY`，也接受 `X-Admin-Key: $ADMIN_API_KEY`。
- 服务端 `ADMIN_API_KEYS` 支持逗号分隔多把，常量时间逐个比较，便于**无停机轮换管理密钥**。
- `ADMIN_API_KEYS` 必填：缺失时服务直接启动失败，管理面绝不裸奔。
- 管理密钥与上传 key 完全隔离：上传 key 不能调管理面，管理密钥也不能用于上传。
- 管理面按 IP 限流，所有写操作写审计。

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
| 签发 | `POST /apps/{appKey}/keys` | 该 app 没有 active key 时创建；已有 active 返回 409 `ACTIVE_KEY_EXISTS`（换 key 用 rotate）；可带 `label`、`expiresAt`（RFC3339） |
| 列表 | `GET /apps/{appKey}/keys` | 全部历史 key（含已吊销），无明文 |
| 轮换 | `POST /keys/{keyId}/rotate` | 同一事务：旧 active → retiring 且 `expires_at = now + graceHours`（默认 24，上限 720）；签发新 active 并返回明文。响应含 `uploadKey`（新，带 `plaintext`）与 `retireKey`（旧） |
| 吊销 | `POST /keys/{keyId}/revoke` | 立即置 revoked；已吊销幂等 200。怀疑泄露时**先吊销再签发**，不要等宽限期 |

轮换流程：`rotate` 拿到新明文 → 立刻更新 GitHub secret `SCHEDULE_UPLOAD_KEY` / `.beta-release.env` →
观察审计旧 key 不再使用（或等宽限期结束）→ 旧 key 自动失效。

## 6. release 下架（yank）语义

接口：`DELETE /api/v1/admin/releases/{releaseId}`（body 可带 `{"reason":"..."}`，已下架幂等 200）、
`POST /api/v1/admin/releases/{releaseId}/restore`。列表用
`GET /api/v1/admin/apps/{appKey}/releases?includeYanked=1&limit=100`，用返回的 **releaseId** 定位。

- yank 只是把 release 的 `status` 从 1 改成 2，latest 查询跳过它；
- 不删除 release 记录与 OSS 对象，可随时 restore，保留审计；
- 同版本同 SHA 重新上传会幂等命中并把该 release 复活为 live；
- **已安装更高版本的用户不会降级**（Android 不允许）。真回滚要发一个 **versionCode 更高** 的包；
- 旧镜像（不认识 release status）会把 yanked 版本当最新返回，回滚镜像前先确认没有 yanked 记录。

## 7. 上传配额与限流

| 机制 | 配置 | 默认 | 触发时 |
|---|---|---|---|
| 每 app 每日新增 release 数 | app 级 `dailyUploadLimit`（未指定用 `DEFAULT_DAILY_UPLOAD_LIMIT`） | 20/自然日 | 429 `UPLOAD_QUOTA_EXCEEDED` |
| 单把上传 key 的令牌桶 | `UPLOAD_RATE_RPS` / `UPLOAD_RATE_BURST` | 0.2 / 3（约 12 次/分钟） | 429 `RATE_LIMITED` |
| 管理面按 IP | `ADMIN_RATE_RPS` / `ADMIN_RATE_BURST` | 0.1 / 5 | 429 `RATE_LIMITED` |
| 公开 latest 接口按 IP | `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` | 2 / 10 | 429 |

- 每日配额按自然日统计，只计「新增 release」；同版本同 SHA 的幂等重传不计数。
- 单文件上限取 `min(app.maxUploadMB, MAX_UPLOAD_MB)`（`app.maxUploadMB=0` 表示沿用全局）。
- 配额/限流按 key/app 维度隔离：beta 渠道的频繁重试不会挤占正式渠道。

调整方式：`PATCH /api/v1/admin/apps/{appKey}` 改 `dailyUploadLimit` / `maxUploadMB` / `allowDowngrade`，无需重启。

## 8. latest 与 app 状态

`GET /api/v1/apps/{appKey}/latest?versionCode=xxx`（公开）：

| 场景 | 行为 |
|---|---|
| app 不存在（未知 appKey） | 200，`hasUpdate=false`、字段为空（官网兼容） |
| app 已停用（status=2） | 404 `APP_NOT_FOUND`（官网回退到内置信息） |
| 正常 app | 只取 `status=live` 的最大 versionCode（yanked 跳过） |
| 无 live release | 200 空 |
| 请求过频 | 429 |

停用/启用：`PATCH /api/v1/admin/apps/{appKey}` body `{"status":2}` / `{"status":1}`。
停用后上传被拒（403 `APP_DISABLED`）、latest 404，历史数据与 key 记录保留。

## 9. 安全注意事项

1. **只存 hash**：数据库/备份泄露无法还原 key。
2. **明文只返回一次**：签发/轮换响应中的 `plaintext` 不要复制进聊天、日志、工单或 git。
3. **管理密钥不进 CI**：`ADMIN_API_KEYS` 只放管理员本机/服务器 `.env`；GitHub Actions 只配 `SCHEDULE_UPLOAD_KEY`。
4. **每渠道/每用途一把 key**：正式与测试渠道分开，CI 与本地脚本分开（`label` 标注用途），
   一把泄露只吊销这一把。
5. **泄露处置**：立即 `POST /keys/{keyId}/revoke`（不要等宽限）→ 签发新 key 并更新使用方 →
   查 `audit_logs` 确认没有异常上传/下架。
6. **轮换**：正式渠道 CI key 建议每 90 天或人员变动时轮换；不要长期保留 retiring key。
7. **截断展示**：日志/界面只展示 `keyId` 与 `last4`，绝不打印完整明文或 `key_hash`。
8. **审计**：所有写操作落 `audit_logs`（`request_id` 与响应头 `X-Request-ID` 对应）；
   `detail` 只放元信息，不放明文与文件内容。
9. **停用与复用**：app 停用后 `app_key` 永久保留，不能给别的应用复用；确需新渠道请新建 appKey。

## 10. 常见运维问答

**Q：上传返回 `APP_KEY_MISMATCH`？**
A：key 属于 A 渠道但表单传了 B 的 appKey。用对应渠道的 key，或省略 `appKey`。

**Q：上传返回 `INVALID_API_KEY`？**
A：确认 CI secret 是 `SCHEDULE_UPLOAD_KEY` 且为本渠道 key；确认 key 未过期/未被吊销
（`GET /apps/{appKey}/keys` 看状态），必要时 `rotate` 后更新 secret。

**Q：某个版本被 yank 了，已升级的用户为什么还在用？**
A：yank 只影响未升级用户的 latest；已升级用户要靠更高 versionCode 的新包覆盖。

**Q：误建了 app / 误发了 key？**
A：app 无删除接口，可 `PATCH status=2` 停用；key 可 revoke。误发的 key 立即 revoke，
同 app 没有其它 active 时再签发新 key。

**Q：管理面返回 `ADMIN_UNAUTHORIZED`？**
A：检查请求头格式与 `.env` 的 `ADMIN_API_KEYS`；多把时任一匹配即可；改 `.env` 后需重启容器。
