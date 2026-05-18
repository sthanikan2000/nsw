CREATE TABLE IF NOT EXISTS company_records (
    id          varchar(100)             NOT NULL PRIMARY KEY,
    name        varchar(255)             NOT NULL,
    ou_id       varchar(255)             NOT NULL UNIQUE,
    ou_handle   varchar(255)             NOT NULL UNIQUE,
    data        jsonb                    NOT NULL DEFAULT '{}',
    created_at  timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at  timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_company_records_ou_id ON company_records (ou_id);
CREATE INDEX idx_company_records_ou_handle ON company_records (ou_handle);

INSERT INTO company_records (id, name, ou_id, ou_handle, data) VALUES
    ('abcd-traders', 'ABCD Traders', 'abcd-traders-id', 'abcd-traders', '{}');
