-- Migration: Create lynk_transactions table
CREATE TABLE lynk_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(150) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    amount INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_lynk_transactions_transaction_id ON lynk_transactions(transaction_id);
CREATE INDEX idx_lynk_transactions_email ON lynk_transactions(email);
