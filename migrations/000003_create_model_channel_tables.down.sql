-- 按依赖反向删除，避免外键约束阻止回滚。
DROP TABLE IF EXISTS channel_models;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS providers;