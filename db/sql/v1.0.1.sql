-- Description: 容器平台引入租户环境概念，租户环境映射集群资源代替原来的租户
-- Date: 2018-08-20 15:00:00
-- Version: 1.0.1

-- 1、创建租户环境表
-- region.tenant_envs definition

CREATE TABLE `tenant_envs` (
  `ID` int unsigned NOT NULL AUTO_INCREMENT,
  `create_time` datetime DEFAULT NULL,
  `name` varchar(40) DEFAULT NULL,
  `uuid` varchar(33) DEFAULT NULL,
  `tenant_id` varchar(255) DEFAULT NULL,
  `tenant_name` varchar(255) DEFAULT NULL,
  `limit_memory` int DEFAULT NULL,
  `status` varchar(255) DEFAULT 'normal',
  `namespace` varchar(32) DEFAULT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `uix_tenant_envs_namespace` (`namespace`),
  UNIQUE KEY `uix_tenant_envs_uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 2. 住户环境表数据迁移
-- 数据导入：tenant => tenant_env
INSERT INTO tenant_envs (
    ID,
    create_time, 
    name, 
    uuid, 
    tenant_id, 
    tenant_name, 
    limit_memory, 
    status, 
    namespace
) SELECT
    ID,
    create_time, 
    name, 
    uuid, 
    uuid, 
    name, 
    limit_memory, 
    status, 
    namespace
FROM tenants;

-- 3. 租户环境相关表
-- 表名的调整：tenant_* => tenant_env_*
RENAME TABLE tenant_lb_mapping_port TO tenant_env_lb_mapping_port;
RENAME TABLE tenant_plugin TO tenant_env_plugin;
RENAME TABLE tenant_plugin_build_version TO tenant_env_plugin_build_version;
RENAME TABLE tenant_plugin_version_config TO tenant_env_plugin_version_config;
RENAME TABLE tenant_plugin_version_env TO tenant_env_plugin_version_env;
RENAME TABLE tenant_service_3rd_party_discovery_cfg TO tenant_env_service_3rd_party_discovery_cfg;
RENAME TABLE tenant_service_3rd_party_endpoints TO tenant_env_service_3rd_party_endpoints;
RENAME TABLE tenant_service_config_file TO tenant_env_service_config_file;
RENAME TABLE tenant_service_plugin_relation TO tenant_env_service_plugin_relation;
RENAME TABLE tenant_service_version TO tenant_env_service_version;
RENAME TABLE tenant_services TO tenant_env_services;
RENAME TABLE tenant_services_autoscaler_rule_metrics TO tenant_env_services_autoscaler_rule_metrics;
RENAME TABLE tenant_services_autoscaler_rules TO tenant_env_services_autoscaler_rules;
RENAME TABLE tenant_services_codecheck TO tenant_env_services_codecheck;
RENAME TABLE tenant_services_delete TO tenant_env_services_delete;
RENAME TABLE tenant_services_envs TO tenant_env_services_envs;
RENAME TABLE tenant_services_event TO tenant_env_services_event;
RENAME TABLE tenant_services_label TO tenant_env_services_label;
RENAME TABLE tenant_services_mnt_relation TO tenant_env_services_mnt_relation;
RENAME TABLE tenant_services_monitor TO tenant_env_services_monitor;
RENAME TABLE tenant_services_port TO tenant_env_services_port;
RENAME TABLE tenant_services_probe TO tenant_env_services_probe;
RENAME TABLE tenant_services_relation TO tenant_env_services_relation;
RENAME TABLE tenant_services_scaling_records TO tenant_env_services_scaling_records;
RENAME TABLE tenant_services_source TO tenant_env_services_source;
RENAME TABLE tenant_services_stream_plugin_port TO tenant_env_services_stream_plugin_port;
RENAME TABLE tenant_services_volume TO tenant_env_services_volume;
RENAME TABLE tenant_services_volume_type TO tenant_env_services_volume_type;

-- 4. 租户环境相关表
-- 表字段名的调整：tenant_id => tenant_env_id，tenant_name => tenant_env_name
ALTER TABLE region.applications CHANGE COLUMN tenant_id tenant_env_id varchar(255);
ALTER TABLE region.tenant_env_plugin CHANGE COLUMN tenant_id tenant_env_id varchar(255);
ALTER TABLE region.tenant_env_services CHANGE COLUMN tenant_id tenant_env_id varchar(32);
ALTER TABLE region.tenant_env_services_delete CHANGE COLUMN tenant_id tenant_env_id varchar(32);
ALTER TABLE region.tenant_env_services_envs CHANGE COLUMN tenant_id tenant_env_id varchar(32);
ALTER TABLE region.tenant_env_services_event CHANGE COLUMN tenant_id tenant_env_id varchar(40);
ALTER TABLE region.tenant_env_services_mnt_relation CHANGE COLUMN tenant_id tenant_env_id varchar(32);
ALTER TABLE region.tenant_env_services_monitor CHANGE COLUMN tenant_id tenant_env_id varchar(40);
ALTER TABLE region.tenant_env_services_port CHANGE COLUMN tenant_id tenant_env_id varchar(32);
ALTER TABLE region.tenant_env_services_relation CHANGE COLUMN tenant_id tenant_env_id varchar(32);

-- tmp
