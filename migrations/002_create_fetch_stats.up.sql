CREATE TABLE IF NOT EXISTS fetch_stats (
    id SERIAL PRIMARY KEY,
    projects_count INTEGER NOT NULL,
    batch_size INTEGER NOT NULL,
    duration_seconds DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
