-- v4: add encrypted function secrets per app
-- actual DDL is applied programmatically in migration.go:migrateAddFunctionSecrets
-- because SQLite has no dynamic SQL for iterating app schemas
SELECT 1;
