#!/bin/bash
# init-db.sh

echo "Waiting for CockroachDB nodes to start..."
sleep 5 

echo "Bootstrapping the Raft Cluster..."
/cockroach/cockroach init --insecure --host=roach1:26257 || true

echo "Applying Database Schema..."
/cockroach/cockroach sql --insecure --host=roach1:26257 -e "
CREATE DATABASE IF NOT EXISTS twitter;
USE twitter;

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now()
);

-- The crucial index for cursor pagination
CREATE INDEX IF NOT EXISTS idx_messages_cursor ON messages (created_at DESC, id DESC);
"

echo "Database initialization complete!"