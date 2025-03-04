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
-- ) PARTITION BY RANGE (created_at); -- we could also introduce time based partitioning here
);

-- Indices
CREATE INDEX idx_closed_bills_user_id ON closed_bills(user_id);
CREATE INDEX idx_closed_bills_status ON closed_bills(status);
CREATE INDEX idx_closed_bills_created_at ON closed_bills(created_at);
