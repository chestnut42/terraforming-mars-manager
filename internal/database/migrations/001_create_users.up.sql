CREATE TABLE users (
    id              TEXT NOT NULL CHECK (id != ''),
    nickname        TEXT NOT NULL,

    created_at      TIMESTAMP WITH TIME ZONE NOT NULL,

    PRIMARY KEY(id)
);

CREATE UNIQUE INDEX uniq_nickname ON users(nickname);
