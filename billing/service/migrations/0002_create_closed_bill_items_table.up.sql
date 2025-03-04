CREATE TABLE closed_bills_items (
    id          UUID PRIMARY KEY,
    bill_id      UUID NOT NULL REFERENCES closed_bills(ID) ON DELETE CASCADE,
    item_id     UUID NOT NULL,
    description TEXT NOT NULL, -- Todo: description type?
    quantity    INT NOT NULL CHECK (Quantity > 0),
    unit_price   DECIMAL(18, 4) NOT NULL,
    currency    CHAR(3) NOT NULL,
    UNIQUE (bill_id, item_id)
); --Todo: add created at, updated at for items
