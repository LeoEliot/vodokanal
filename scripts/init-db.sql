-- Vodokanal Database Schema
-- Initial migration script

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- USERS & AUTH
-- ============================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'subscriber',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- ============================================
-- SUBSCRIBERS
-- ============================================

CREATE TABLE IF NOT EXISTS subscribers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    account_number VARCHAR(50) UNIQUE NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    email VARCHAR(255),
    phone VARCHAR(20),
    address VARCHAR(500) NOT NULL,
    apartment VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_subscribers_account ON subscribers(account_number);
CREATE INDEX idx_subscribers_user ON subscribers(user_id);

-- ============================================
-- COUNTERS (WATER METERS)
-- ============================================

CREATE TABLE IF NOT EXISTS counters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscriber_id UUID NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    serial_number VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('hot', 'cold', 'sewage')),
    brand VARCHAR(100),
    model VARCHAR(100),
    installation_date DATE NOT NULL,
    verification_date DATE NOT NULL,
    next_verification_date DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(subscriber_id, serial_number)
);

CREATE INDEX idx_counters_subscriber ON counters(subscriber_id);
CREATE INDEX idx_counters_type ON counters(type);

-- ============================================
-- READINGS
-- ============================================

CREATE TABLE IF NOT EXISTS readings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscriber_id UUID NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    counter_id UUID NOT NULL REFERENCES counters(id) ON DELETE CASCADE,
    value DECIMAL(10, 3) NOT NULL,
    previous_value DECIMAL(10, 3),
    consumption DECIMAL(10, 3),
    date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'verified', 'rejected')),
    verified_by UUID REFERENCES users(id),
    verified_at TIMESTAMP,
    rejection_reason TEXT,
    photo_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_readings_subscriber ON readings(subscriber_id);
CREATE INDEX idx_readings_counter ON readings(counter_id);
CREATE INDEX idx_readings_date ON readings(date);
CREATE INDEX idx_readings_status ON readings(status);

-- ============================================
-- TARIFFS
-- ============================================

CREATE TABLE IF NOT EXISTS tariffs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    hot_water_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
    cold_water_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
    sewage_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
    service_charge DECIMAL(10, 2) NOT NULL DEFAULT 0,
    valid_from DATE NOT NULL,
    valid_to DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tariffs_active ON tariffs(is_active);
CREATE INDEX idx_tariffs_valid ON tariffs(valid_from, valid_to);

-- ============================================
-- BILLS
-- ============================================

CREATE TABLE IF NOT EXISTS bills (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscriber_id UUID NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    account_number VARCHAR(50) NOT NULL,
    bill_number VARCHAR(100) UNIQUE NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    due_date DATE NOT NULL,

    -- Usage
    hot_water_usage DECIMAL(10, 3) NOT NULL DEFAULT 0,
    cold_water_usage DECIMAL(10, 3) NOT NULL DEFAULT 0,

    -- Tariffs (snapshot at billing time)
    hot_water_tariff DECIMAL(10, 2) NOT NULL,
    cold_water_tariff DECIMAL(10, 2) NOT NULL,

    -- Amounts
    hot_water_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
    cold_water_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
    sewage_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
    service_charge DECIMAL(10, 2) NOT NULL DEFAULT 0,
    maintenance_charge DECIMAL(10, 2) NOT NULL DEFAULT 0,
    total_amount DECIMAL(10, 2) NOT NULL,

    -- Status
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'issued', 'paid', 'overdue', 'cancelled')),
    paid_at TIMESTAMP,
    payment_id VARCHAR(255),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bills_subscriber ON bills(subscriber_id);
CREATE INDEX idx_bills_account ON bills(account_number);
CREATE INDEX idx_bills_period ON bills(period_start, period_end);
CREATE INDEX idx_bills_status ON bills(status);
CREATE INDEX idx_bills_due_date ON bills(due_date);

-- ============================================
-- PAYMENTS
-- ============================================

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id UUID NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
    subscriber_id UUID NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'RUB',
    method VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    gateway VARCHAR(50),
    gateway_tx_id VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    failed_at TIMESTAMP,
    refunded_amount DECIMAL(10, 2) DEFAULT 0,
    refund_reason TEXT
);

CREATE INDEX idx_payments_bill ON payments(bill_id);
CREATE INDEX idx_payments_subscriber ON payments(subscriber_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_gateway_tx ON payments(gateway_tx_id);

-- ============================================
-- TICKETS
-- ============================================

CREATE TABLE IF NOT EXISTS tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscriber_id UUID REFERENCES subscribers(id) ON DELETE SET NULL,
    account_number VARCHAR(50),
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL CHECK (category IN ('leak', 'repair', 'billing', 'quality', 'other')),
    priority VARCHAR(20) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed', 'cancelled')),

    -- Address
    address VARCHAR(500),
    apartment VARCHAR(20),

    -- Assignment
    assigned_to UUID,
    assigned_at TIMESTAMP,

    -- Resolution
    resolution TEXT,
    resolved_at TIMESTAMP,
    resolved_by UUID,

    -- Scheduling
    scheduled_date DATE,
    scheduled_time VARCHAR(20),

    -- Contact
    contact_name VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    contact_email VARCHAR(255),

    -- Feedback
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    feedback TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    source VARCHAR(20) DEFAULT 'web'
);

CREATE INDEX idx_tickets_subscriber ON tickets(subscriber_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_category ON tickets(category);
CREATE INDEX idx_tickets_assigned ON tickets(assigned_to);

-- ============================================
-- TICKET COMMENTS
-- ============================================

CREATE TABLE IF NOT EXISTS ticket_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    author_id UUID NOT NULL REFERENCES users(id),
    author_name VARCHAR(255) NOT NULL,
    author_role VARCHAR(50) NOT NULL,
    is_internal BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ticket_comments_ticket ON ticket_comments(ticket_id);

-- ============================================
-- TICKET HISTORY
-- ============================================

CREATE TABLE IF NOT EXISTS ticket_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    field VARCHAR(50) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by UUID NOT NULL REFERENCES users(id),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ticket_history_ticket ON ticket_history(ticket_id);

-- ============================================
-- EMPLOYEES
-- ============================================

CREATE TABLE IF NOT EXISTS employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    role VARCHAR(50) NOT NULL CHECK (role IN ('manager', 'operator', 'technician')),
    is_active BOOLEAN DEFAULT true,
    skills TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_employees_active ON employees(is_active);
CREATE INDEX idx_employees_role ON employees(role);

-- ============================================
-- NOTIFICATIONS
-- ============================================

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    subscriber_id UUID REFERENCES subscribers(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('email', 'sms', 'push')),
    channel VARCHAR(50) NOT NULL,
    subject VARCHAR(500),
    body TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    sent_at TIMESTAMP,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_subscriber ON notifications(subscriber_id);
CREATE INDEX idx_notifications_status ON notifications(status);

-- ============================================
-- FUNCTIONS & TRIGGERS
-- ============================================

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to tables with updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscribers_updated_at BEFORE UPDATE ON subscribers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_counters_updated_at BEFORE UPDATE ON counters
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_readings_updated_at BEFORE UPDATE ON readings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bills_updated_at BEFORE UPDATE ON bills
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tickets_updated_at BEFORE UPDATE ON tickets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- SEED DATA (for development)
-- ============================================

-- Default admin user (password: admin123)
INSERT INTO users (id, email, password_hash, name, role) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@vodokanal.ru',
     '$2a$10$YourHashedPasswordHere', 'Administrator', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Default tariff
INSERT INTO tariffs (name, hot_water_price, cold_water_price, sewage_price, service_charge, valid_from, is_active) VALUES
    ('Базовый тариф 2024', 150.50, 45.20, 30.00, 350.00, '2024-01-01', true)
ON CONFLICT DO NOTHING;

-- ============================================
-- VIEWS
-- ============================================

-- View for subscriber with latest reading info
CREATE OR REPLACE VIEW v_subscriber_readings AS
SELECT
    s.id,
    s.account_number,
    s.last_name,
    s.first_name,
    s.middle_name,
    s.email,
    s.phone,
    s.address,
    COUNT(DISTINCT c.id) as counter_count,
    COUNT(r.id) FILTER (WHERE r.date = (SELECT MAX(date) FROM readings r2 WHERE r2.counter_id = r.counter_id)) as current_readings
FROM subscribers s
LEFT JOIN counters c ON c.subscriber_id = s.id AND c.is_active = true
LEFT JOIN readings r ON r.counter_id = c.id
GROUP BY s.id;
