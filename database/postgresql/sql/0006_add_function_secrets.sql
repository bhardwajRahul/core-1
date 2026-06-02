DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT lower(name) AS name FROM sb.apps LOOP
        EXECUTE format('
            ALTER TABLE %I.sb_functions
            ADD COLUMN IF NOT EXISTS function_secrets BYTEA
        ', r.name);
    END LOOP;
END $$;
