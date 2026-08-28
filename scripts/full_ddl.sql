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



-- end check
SELECT 'FULL_DDL_OK' AS status;

