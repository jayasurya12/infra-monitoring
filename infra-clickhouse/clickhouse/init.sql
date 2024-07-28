CREATE DATABASE IF NOT EXISTS infra_monitoring;

USE infra_monitoring;

CREATE TABLE IF NOT EXISTS infra_monitoring.system_info (
    id UInt32,
    info String,
    cpu_usage Float64,
    memory_usage Float64,
    disk_usage String, -- Add this line to include the disk usage as a JSON string
    processes Array(String),
    connections Array(String),
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id
