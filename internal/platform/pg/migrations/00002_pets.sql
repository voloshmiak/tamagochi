-- +goose Up
CREATE TABLE pets (
    id           text             PRIMARY KEY,
    owner_id     text             NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    name         text             NOT NULL CHECK (char_length(name) BETWEEN 2 AND 20),
    appearance   text             NOT NULL,
    satiety      double precision NOT NULL CHECK (satiety BETWEEN 0 AND 100),
    mood         double precision NOT NULL CHECK (mood BETWEEN 0 AND 100),
    energy       double precision NOT NULL CHECK (energy BETWEEN 0 AND 100),
    cleanliness  double precision NOT NULL CHECK (cleanliness BETWEEN 0 AND 100),
    born_at      timestamptz      NOT NULL,
    updated_at   timestamptz      NOT NULL,
    last_actions jsonb            NOT NULL DEFAULT '{}'
);

CREATE TABLE pet_traits (
    pet_id text             NOT NULL REFERENCES pets (id) ON DELETE CASCADE,
    trait  text             NOT NULL,
    weight double precision NOT NULL CHECK (weight BETWEEN 0 AND 1),
    PRIMARY KEY (pet_id, trait)
);

CREATE TABLE items (
    id   text PRIMARY KEY,
    name text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('food', 'toy', 'soap'))
);

INSERT INTO items (id, name, kind) VALUES
    ('apple', 'Apple', 'food'),
    ('fish',  'Fish',  'food'),
    ('ball',  'Ball',  'toy'),
    ('soap',  'Soap',  'soap');

CREATE TABLE inventories (
    owner_id        text        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    daily_reward_at timestamptz
);

CREATE TABLE inventory_items (
    owner_id text    NOT NULL REFERENCES inventories (owner_id) ON DELETE CASCADE,
    item_id  text    NOT NULL REFERENCES items (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (owner_id, item_id)
);

-- +goose Down
DROP TABLE inventory_items;
DROP TABLE inventories;
DROP TABLE items;
DROP TABLE pet_traits;
DROP TABLE pets;
