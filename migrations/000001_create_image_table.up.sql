CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    status TEXT NOT NULL DEFAULT 'created'
        CHECK (status IN ('created', 'processing', 'processed', 'deleted')),

    format TEXT NOT NULL,
    name TEXT NOT NULL,

    watermark TEXT CHECK (char_length(watermark) <= 20),

    resize_height INT,
    resize_width INT
);