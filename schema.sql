CREATE DATABASE IF NOT EXISTS roguelite;
USE roguelite;

CREATE TABLE IF NOT EXISTS players (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    x INT DEFAULT 0,
    y INT DEFAULT 0,
    z INT DEFAULT 0,
    hp INT DEFAULT 10,
    max_hp INT DEFAULT 10,
    inventory TEXT DEFAULT '[]',
    weapon VARCHAR(255) DEFAULT '',
    armor VARCHAR(255) DEFAULT '',
    coins INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS levels (
    z INT PRIMARY KEY,
    map_data TEXT
);