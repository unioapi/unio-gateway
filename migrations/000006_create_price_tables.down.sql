-- price_snapshots 依赖 prices 和 request_records，回滚时先删除快照表。
DROP TABLE IF EXISTS price_snapshots;
DROP TABLE IF EXISTS prices;
