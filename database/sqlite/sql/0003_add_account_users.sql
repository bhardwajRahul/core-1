-- v3: add sb_account_users per-app table
-- actual DDL is applied programmatically in migration.go:migrateAddAccountUsers
-- because SQLite has no dynamic SQL for iterating app schemas
SELECT 1;
