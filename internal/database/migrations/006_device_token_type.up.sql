ALTER TABLE manager_users
    ADD COLUMN device_token_type TEXT NOT NULL default 'prod';
