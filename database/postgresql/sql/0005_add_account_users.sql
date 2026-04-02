DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT lower(name) AS name FROM sb.apps LOOP
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.sb_account_users (
                id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
                user_id    uuid NOT NULL REFERENCES %I.sb_tokens(id)   ON DELETE CASCADE,
                account_id uuid NOT NULL REFERENCES %I.sb_accounts(id) ON DELETE CASCADE,
                email      TEXT NOT NULL,
                role       INTEGER NOT NULL DEFAULT 0,
                token      TEXT NOT NULL UNIQUE,
                created    TIMESTAMP NOT NULL,
                UNIQUE(user_id, account_id)
            )', r.name, r.name, r.name);
    END LOOP;
END $$;
