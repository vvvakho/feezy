CREATE TABLE open_bills (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    request_id UUID UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
