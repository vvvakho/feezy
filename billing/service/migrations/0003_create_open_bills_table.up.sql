CREATE TABLE open_bills (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    request_id UUID UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indices
CREATE INDEX idx_open_bills_user_id ON open_bills(user_id);
CREATE INDEX idx_open_bills_status ON open_bills(status);
CREATE INDEX idx_open_bills_created_at ON open_bills(created_at);
