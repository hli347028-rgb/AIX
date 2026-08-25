
/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `aix_prices` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `price` decimal(36,18) NOT NULL,
  `effective_date` date NOT NULL,
  `remark` varchar(255) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_aix_prices_effective_date` (`effective_date`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `exchange_records` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL,
  `from_asset` varchar(16) NOT NULL,
  `from_amount` decimal(36,18) NOT NULL,
  `to_asset` varchar(16) NOT NULL,
  `to_amount` decimal(36,18) NOT NULL,
  `exchange_price` decimal(36,18) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'completed',
  `remark` varchar(255) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  `fee_amount` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `fee_rate` decimal(12,6) NOT NULL DEFAULT 0.000000,
  PRIMARY KEY (`id`),
  KEY `idx_exchange_records_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mgmt_rewards` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL,
  `from_user_id` bigint(20) NOT NULL,
  `source_order_id` bigint(20) NOT NULL,
  `base_amount` decimal(36,18) NOT NULL,
  `rate` decimal(36,18) NOT NULL,
  `total_amount` decimal(36,18) NOT NULL,
  `released_amount` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `created_time` datetime(3) DEFAULT NULL,
  `updated_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_mgmt_source` (`user_id`,`source_order_id`),
  KEY `idx_mgmt_rewards_user_id` (`user_id`),
  KEY `idx_mgmt_rewards_from_user_id` (`from_user_id`),
  KEY `idx_mgmt_rewards_source_order_id` (`source_order_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `orders` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `principal` decimal(36,18) NOT NULL,
  `exit_cap` decimal(36,18) NOT NULL,
  `earned_total` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `direct_base` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `from_recharge` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `from_reward` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `fund_source` varchar(16) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'active',
  `exited_time` datetime(3) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  `updated_time` datetime(3) DEFAULT NULL,
  `from_win` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `points` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  PRIMARY KEY (`id`),
  KEY `idx_orders_user_status` (`user_id`,`status`),
  KEY `idx_orders_status_created` (`status`,`created_time`),
  KEY `idx_orders_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `recharges` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `amount` decimal(36,18) NOT NULL,
  `tx_hash` varchar(66) NOT NULL,
  `from_address` varchar(42) DEFAULT NULL,
  `to_address` varchar(42) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `confirmed_time` datetime(3) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  `updated_time` datetime(3) DEFAULT NULL,
  `message` text DEFAULT NULL,
  `expire_at` datetime(3) DEFAULT NULL,
  `asset` varchar(16) NOT NULL DEFAULT 'USDT',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_recharges_tx_hash` (`tx_hash`),
  KEY `idx_recharges_user_status` (`user_id`,`status`),
  KEY `idx_recharges_user_id` (`user_id`),
  KEY `idx_recharges_asset` (`asset`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `reward_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `from_user_id` bigint(20) unsigned DEFAULT NULL,
  `order_id` bigint(20) unsigned DEFAULT NULL,
  `batch_id` bigint(20) unsigned DEFAULT NULL,
  `type` varchar(32) NOT NULL,
  `asset` varchar(16) NOT NULL,
  `amount` decimal(36,18) NOT NULL,
  `base_amount` decimal(36,18) DEFAULT NULL,
  `rate` decimal(36,18) DEFAULT NULL,
  `exit_applied` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `meta` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`meta`)),
  `settlement_date` date DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_reward_user_type_created` (`user_id`,`type`,`created_time`),
  KEY `idx_reward_batch` (`batch_id`),
  KEY `idx_reward_order` (`order_id`),
  KEY `idx_reward_settlement_date` (`settlement_date`),
  KEY `idx_reward_logs_user_id` (`user_id`),
  KEY `idx_reward_logs_order_id` (`order_id`),
  KEY `idx_reward_logs_batch_id` (`batch_id`),
  KEY `idx_reward_logs_settlement_date` (`settlement_date`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `settings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(64) NOT NULL,
  `value` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`value`)),
  `created_time` datetime(3) DEFAULT NULL,
  `updated_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_settings_key` (`key`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `settlement_batches` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `settlement_date` date NOT NULL,
  `aix_price` decimal(36,18) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'running',
  `static_count` int(10) unsigned NOT NULL DEFAULT 0,
  `static_amount` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `mgmt_count` int(10) unsigned NOT NULL DEFAULT 0,
  `mgmt_amount` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `started_time` datetime(3) DEFAULT NULL,
  `finished_time` datetime(3) DEFAULT NULL,
  `error_msg` varchar(512) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_settlement_batches_settlement_date` (`settlement_date`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `transfers` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `from_user_id` bigint(20) unsigned NOT NULL,
  `to_user_id` bigint(20) unsigned NOT NULL,
  `asset` varchar(16) NOT NULL,
  `amount` decimal(36,18) NOT NULL,
  `pay_from` varchar(16) DEFAULT NULL,
  `from_recharge_debit` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `from_reward_debit` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `to_credit_reward` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `to_credit_aix` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `remark` varchar(255) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_transfers_from_created` (`from_user_id`,`created_time`),
  KEY `idx_transfers_to_created` (`to_user_id`,`created_time`),
  KEY `idx_transfers_from_user_id` (`from_user_id`),
  KEY `idx_transfers_to_user_id` (`to_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `address` varchar(42) NOT NULL,
  `inviter_id` bigint(20) unsigned DEFAULT NULL,
  `invite_code` varchar(64) NOT NULL,
  `usdt_recharge` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `usdt_reward` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `aix_balance` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `mgmt_level` int(11) NOT NULL DEFAULT 0,
  `small_area_perf` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `team_perf` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `status` int(11) NOT NULL DEFAULT 1,
  `created_time` datetime(3) DEFAULT NULL,
  `updated_time` datetime(3) DEFAULT NULL,
  `role` varchar(16) NOT NULL DEFAULT 'user',
  `static_usdt_total` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `large_area_perf` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `mgmt_level_locked` tinyint(1) NOT NULL DEFAULT 0,
  `win_balance` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `win_recharge_balance` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `pending_mgmt_reward` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `overflow_reward` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `overflow_direct` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `points` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `points_all` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `is_zero_account` tinyint(1) NOT NULL DEFAULT 0,
  `is_community_subsidy` tinyint(1) NOT NULL DEFAULT 0,
  `zero_account_set_at` datetime(3) DEFAULT NULL,
  `community_subsidy_set_at` datetime(3) DEFAULT NULL,
  `zero_account_reward_total` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `community_subsidy_total` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_address` (`address`),
  UNIQUE KEY `idx_users_invite_code` (`invite_code`),
  KEY `idx_users_inviter_id` (`inviter_id`),
  KEY `idx_users_mgmt_level` (`mgmt_level`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `win_prices` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `price` decimal(36,18) NOT NULL,
  `source` varchar(32) NOT NULL DEFAULT 'oracle',
  `updated_time` datetime(3) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `withdrawals` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL,
  `asset` varchar(16) NOT NULL DEFAULT 'AIX',
  `amount` decimal(36,18) NOT NULL,
  `fee` decimal(36,18) NOT NULL DEFAULT 0.000000000000000000,
  `pay_amount` decimal(36,18) NOT NULL,
  `to_address` varchar(42) NOT NULL,
  `tx_hash` varchar(66) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `remark` varchar(255) DEFAULT NULL,
  `created_time` datetime(3) DEFAULT NULL,
  `updated_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_withdrawals_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

