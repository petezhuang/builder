-- 该文件用于创建数据库表结构
use spx;

if exists (select * from information_schema.tables where table_name = 'aiResource') then
    drop table aiResource;
end if;
if exists (select * from information_schema.tables where table_name = 'label') then
    drop table label;
end if;
if exists (select * from information_schema.tables where table_name = 'resource_label') then
    drop table resource_label;
end if;

CREATE TABLE aiResource (
                            aiResourceId BIGINT PRIMARY KEY AUTO_INCREMENT,
                            url VARCHAR(255) NOT NULL,
                            createTime DATETIME DEFAULT CURRENT_TIMESTAMP,
    -- 其他冗余字段...
);

CREATE TABLE label (
                       labelId BIGINT PRIMARY KEY AUTO_INCREMENT,
                       labelName VARCHAR(50) UNIQUE NOT NULL
);
CREATE TABLE resource_label (
                                id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                aiResourceId BIGINT NOT NULL,
                                labelId BIGINT NOT NULL,
    -- 防止重复
);