-- 001_per_app_keys.sql —— per-app 上传密钥 / 审计日志 / release 下架（MySQL 8.0）
--
-- 用途：生产环境显式升级参考（先备份，再执行；便于人工审阅每一步变更）。
-- 服务启动时 GORM AutoMigrate 会做等价的增量变更（新表 + 新列，全部带默认值，
-- 存量行可安全回填），因此本文件不是必须执行；但生产首次升级建议执行一次：
--   mysqldump -u<user> -p<pass> --single-transaction <db> > backup_before_001.sql
--   mysql -u<user> -p<pass> <db> < migrations/001_per_app_keys.sql
--
-- 注意：
--   1. MySQL 8.0 的 ADD COLUMN / ADD INDEX 不支持 IF NOT EXISTS，列或索引已存在会报
--      1060/1061；若你已由 AutoMigrate 建好，跳过对应语句即可。
--   2. 老实例的 apps 记录（schedule / schedule-beta 等）会自动获得新列默认值；
--      升级到 dual 模式后，用 admin API 给它们各签发一把 key 即可。
--   3. app_upload_keys 每个 app 同一时刻最多一把 status=1(active)，该不变量由
--      应用层在事务里锁 apps 行保证，数据库不建唯一索引（轮换期需要 active+retiring 并存）。

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------------
-- 1. 新表：每应用上传密钥
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS app_upload_keys (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  app_id BIGINT UNSIGNED NOT NULL COMMENT 'apps.id（应用层保证存在）',
  key_id VARCHAR(8) NOT NULL COMMENT '明文 key 中的公开标识（8 位 hex）',
  key_hash VARCHAR(64) NOT NULL COMMENT 'SHA256(完整明文 key)，明文不落库',
  last4 VARCHAR(4) NOT NULL COMMENT '仅用于展示',
  label VARCHAR(64) NOT NULL COMMENT '用途标签，如 gha-release / local-beta',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '1 active 2 retiring(轮换宽限) 3 revoked',
  expires_at DATETIME(3) NULL COMMENT '为空表示不过期',
  last_used_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  revoked_at DATETIME(3) NULL,
  revoke_reason VARCHAR(255) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_app_upload_keys_key_id (key_id),
  UNIQUE KEY uk_app_upload_keys_key_hash (key_hash),
  KEY idx_app_upload_keys_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='每应用上传密钥（只存哈希）';

-- ---------------------------------------------------------------------------
-- 2. 新表：操作审计
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  request_id VARCHAR(32) NULL COMMENT '请求 ID，与响应头 X-Request-ID 对应',
  actor_type VARCHAR(16) NOT NULL COMMENT 'admin / app / legacy / anonymous',
  actor_key_id VARCHAR(8) NULL COMMENT 'app 上传时的 key_id',
  action VARCHAR(64) NOT NULL COMMENT 'upload.success / app.create / key.issue ...',
  app_key VARCHAR(64) NULL,
  target_type VARCHAR(32) NULL,
  target_id VARCHAR(64) NULL,
  ip VARCHAR(64) NULL,
  user_agent VARCHAR(255) NULL,
  detail TEXT NULL COMMENT 'JSON 文本',
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_audit_logs_request_id (request_id),
  KEY idx_audit_logs_action (action),
  KEY idx_audit_logs_app_key (app_key),
  KEY idx_audit_logs_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作审计（不含明文密钥）';

-- ---------------------------------------------------------------------------
-- 3. 存量表：apps 补列（已存在则跳过）
-- ---------------------------------------------------------------------------
ALTER TABLE apps
  ADD COLUMN status TINYINT NOT NULL DEFAULT 1 COMMENT '1 active 2 disabled',
  ADD COLUMN max_upload_mb BIGINT NOT NULL DEFAULT 0 COMMENT '0=用全局 MAX_UPLOAD_MB',
  ADD COLUMN daily_upload_limit BIGINT NOT NULL DEFAULT 20 COMMENT '每自然日新增 release 上限',
  ADD COLUMN allow_downgrade TINYINT(1) NOT NULL DEFAULT 0 COMMENT '允许上传较低 versionCode';

-- 老实例缺失的 app 记录可按需补（一般不需要；schedule-beta 是独立渠道记录）。
INSERT INTO apps (app_key, name, platform, status)
VALUES
  ('schedule', '西大课栈', 'android', 1),
  ('schedule-beta', '西大课栈 Beta', 'android', 1)
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- ---------------------------------------------------------------------------
-- 4. 存量表：releases 补列 + latest 查询索引（已存在则跳过）
-- ---------------------------------------------------------------------------
ALTER TABLE releases
  ADD COLUMN status TINYINT NOT NULL DEFAULT 1 COMMENT '1 live 2 yanked',
  ADD COLUMN upload_key_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'app_upload_keys.id；legacy 为 0',
  ADD COLUMN yanked_at DATETIME(3) NULL,
  ADD COLUMN yanked_by VARCHAR(64) NULL,
  ADD COLUMN yank_reason VARCHAR(255) NULL,
  ADD INDEX idx_app_status_version (app_key, status, version_code);

-- ---------------------------------------------------------------------------
-- 5. 升级后操作（人工，不在本 SQL 内）：
--    1) .env 增加 ADMIN_API_KEYS=<openssl rand -hex 32>，并设 UPLOAD_AUTH_MODE=dual，
--       保留旧 UPLOAD_API_KEY 供过渡；
--    2) 启动新镜像；
--    3) 用 admin API 给 schedule / schedule-beta 各签发一把 key：
--       POST /api/v1/admin/apps/schedule/keys        （已存在 app 直接签发）
--       POST /api/v1/admin/apps/schedule-beta/keys
--    4) 把返回的明文 key 分别写进 GitHub Actions secret（schedule）与
--       .beta-release.env 的 UPDATE_API_KEY（schedule-beta）；
--    5) 观察 audit_logs 中 actor_type=legacy 的用量归零后，切 UPLOAD_AUTH_MODE=scoped，
--       再保留旧 key 一段时间后从 .env 删除。
--    回退：UPLOAD_AUTH_MODE 切回 dual（或 legacy）并回滚旧镜像；新表/新列无破坏性变更。
-- ---------------------------------------------------------------------------
