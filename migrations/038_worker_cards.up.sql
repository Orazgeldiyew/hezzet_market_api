CREATE TABLE worker_cards (
    id         BIGSERIAL PRIMARY KEY,
    worker_id  BIGINT NOT NULL REFERENCES workers(id),
    card_code  TEXT NOT NULL UNIQUE,
    label      TEXT NOT NULL DEFAULT '',
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_worker_cards_card_code ON worker_cards(card_code);
CREATE INDEX idx_worker_cards_worker_id ON worker_cards(worker_id);
