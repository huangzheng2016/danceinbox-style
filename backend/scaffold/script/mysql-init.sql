-- Runs once, when the mysql data directory is first created. The prod database
-- and the app user come from MYSQL_DATABASE / MYSQL_USER in docker-compose.yml.
CREATE DATABASE IF NOT EXISTS MODULE_NAME_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON MODULE_NAME_test.* TO 'app'@'%';
