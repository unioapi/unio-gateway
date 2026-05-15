-- 按依赖反向删除，避免外键约束阻止回滚。
DROP TABLE IF EXISTS request_attempts;
DROP TABLE IF EXISTS request_records;
