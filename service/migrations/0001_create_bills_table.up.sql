-- Create Closed Bills Table
CREATE TABLE closed_bills (
    ID          UUID PRIMARY KEY,
    UserID      UUID NOT NULL,
    Status      VARCHAR(50) NOT NULL,
    TotalAmount DECIMAL(18, 4) NOT NULL,
    Currency    CHAR(3) NOT NULL,
    CreatedAt   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UpdatedAt   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ClosedAt    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create Closed Bill Items Table
CREATE TABLE closed_bills_items (
    ID          UUID PRIMARY KEY,
    BillID      UUID NOT NULL REFERENCES ClosedBills(ID) ON DELETE CASCADE,
    Description TEXT NOT NULL,
    Quantity    INT NOT NULL CHECK (Quantity > 0),
    UnitPrice   DECIMAL(18, 4) NOT NULL,
    Currency    CHAR(3) NOT NULL
);
