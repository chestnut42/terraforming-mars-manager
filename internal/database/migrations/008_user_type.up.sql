ALTER TABLE manager_users
    ADD COLUMN type TEXT NOT NULL default 'blank';

UPDATE manager_users SET type = 'active' WHERE nickname NOT LIKE 'Player %';
