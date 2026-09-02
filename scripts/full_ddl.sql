-- 评论表
CREATE TABLE IF NOT EXISTS `comments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `create_time` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `update_time` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted` datetime(3) DEFAULT NULL COMMENT '删除时间，空表示未删除',
  `post_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `content` text COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_comments_deleted` (`deleted`),
  KEY `idx_comments_post_id` (`post_id`),
  KEY `idx_comments_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 点赞表
CREATE TABLE IF NOT EXISTS `post_likes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `create_time` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `update_time` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `post_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_like_user_post` (`post_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 帖子表
CREATE TABLE IF NOT EXISTS `posts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `create_time` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `update_time` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted` datetime(3) DEFAULT NULL COMMENT '删除时间，空表示未删除',
  `user_id` bigint(20) unsigned NOT NULL,
  `title` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `content` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `visible` tinyint(4) NOT NULL COMMENT '可见性状态，1所有人可见，0仅自己可见',
  `pinned_until` datetime(3) DEFAULT NULL COMMENT '置顶到期时间',
  PRIMARY KEY (`id`),
  KEY `idx_posts_deleted` (`deleted`),
  KEY `idx_posts_user_id` (`user_id`),
  KEY `idx_posts_pinned_until` (`pinned_until`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户信息表
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `create_time` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `update_time` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted` datetime(3) DEFAULT NULL COMMENT '删除时间，空表示未删除',
  `username` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户名',
  `password` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `nickname` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '用户昵称',
  `status` tinyint(4) DEFAULT 1 COMMENT '用户状态，1:正常，0禁言',
  `introduction` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '个人介绍',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_deleted` (`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 邀请码表
CREATE TABLE IF NOT EXISTS `invitations` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT COMMENT '主键ID',
  `create_time` DATETIME(3) NULL COMMENT '创建时间',
  `update_time` DATETIME(3) NULL COMMENT '更新时间',
  `code` CHAR(6) NOT NULL COMMENT '邀请码',
  `creator_id` BIGINT UNSIGNED NOT NULL COMMENT '创建用户ID',
  `used_by` BIGINT UNSIGNED NULL COMMENT '使用用户ID',
  `used_at` DATETIME(3) NULL COMMENT '使用时间',
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_invitations_code` (`code`),
  INDEX `idx_invitations_creator_id` (`creator_id`),
  UNIQUE INDEX `idx_invitations_used_by` (`used_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 给用户表增加注册时使用的邀请码
ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `invite_code` CHAR(6) NULL COMMENT '注册时使用的邀请码' AFTER `introduction`;

-- 给用户邀请码增加唯一索引
ALTER TABLE `users` ADD UNIQUE INDEX IF NOT EXISTS `idx_users_invite_code` (`invite_code`);

-- 给用户表增加头像链接及独立修改时间；空链接恢复默认头像
ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `avatar_url` VARCHAR(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL COMMENT '头像图片链接' AFTER `introduction`;
ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `avatar_updated_at` DATETIME(3) NULL COMMENT '头像最后修改时间' AFTER `avatar_url`;

-- 密码重置时递增会话版本，使旧 JWT 立即失效
ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `session_version` BIGINT(20) UNSIGNED NOT NULL DEFAULT 0 COMMENT '会话版本' AFTER `password`;

-- 用户留言表
CREATE TABLE IF NOT EXISTS `messages` (
  `id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `create_time` DATETIME(3) DEFAULT NULL COMMENT '创建时间',
  `update_time` DATETIME(3) DEFAULT NULL COMMENT '更新时间',
  `deleted` DATETIME(3) DEFAULT NULL COMMENT '删除时间，空表示未删除',
  `user_id` BIGINT(20) UNSIGNED NOT NULL COMMENT '留言用户ID',
  `content` TEXT COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '留言内容',
  PRIMARY KEY (`id`),
  KEY `idx_messages_user_deleted_created` (`user_id`, `deleted`, `create_time`, `id`),
  KEY `idx_messages_deleted_created` (`deleted`, `create_time`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


-- end check
SELECT 'FULL_DDL_OK' AS status;
