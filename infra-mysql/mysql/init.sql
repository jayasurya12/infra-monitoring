CREATE DATABASE IF NOT EXISTS infra_monitoring;

USE infra_monitoring;

CREATE TABLE IF NOT EXISTS system_info (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    info TEXT,
    cpu_usage FLOAT,
    memory_usage FLOAT,
    disk_usage TEXT,
    processes TEXT,
    connections TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
