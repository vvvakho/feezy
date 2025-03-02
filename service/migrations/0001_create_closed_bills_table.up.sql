CREATE TABLE closed_bills (
    id          UUID PRIMARY KEY,
    user_id      UUID NOT NULL,
    status      VARCHAR(50) NOT NULL,
    total_amount DECIMAL(18, 4) NOT NULL,
    currency    CHAR(3) NOT NULL,
    request_id UUID UNIQUE,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
