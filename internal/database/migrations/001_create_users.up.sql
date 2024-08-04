CREATE TABLE manager_users (
    id              TEXT NOT NULL CHECK (id != ''),
    nickname        TEXT NOT NULL,
    color           TEXT NOT NULL,

    created_at      TIMESTAMP WITH TIME ZONE NOT NULL,

    PRIMARY KEY(id)
);

CREATE UNIQUE INDEX idx_uniq_nickname ON manager_users(nickname);
