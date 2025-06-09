#!/bin/bash

# generate_test_db.sh - Create a test chat.db with dummy data for unit tests
# This script creates a minimal chat.db with the same schema as the real one
# but contains only dummy data for testing purposes.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REAL_DB="$HOME/Library/Messages/chat.db"
TEST_DIR="$PROJECT_ROOT/internal/archiver/testdata"
TEST_DB="$TEST_DIR/chat.db"
SCHEMA_FILE="$PROJECT_ROOT/schema.sql"

echo "🔧 Generating test chat.db for unit tests..."

# Check if real chat.db exists
if [[ ! -f "$REAL_DB" ]]; then
    echo "❌ Error: Real chat.db not found at $REAL_DB"
    echo "   This script requires access to the system's iMessage database."
    exit 1
fi

# Create test directory
mkdir -p "$TEST_DIR"

# Remove existing test database if it exists
if [[ -f "$TEST_DB" ]]; then
    echo "🗑️  Removing existing test database..."
    rm "$TEST_DB"
fi

# Export schema from real database, filtering out sqlite_* internals
echo "📤 Exporting schema from real chat.db..."
sqlite3 "$REAL_DB" .schema \
    | grep -vE 'CREATE (TABLE|INDEX) (sqlite_sequence|sqlite_stat[0-9]*)' \
    > "$SCHEMA_FILE"

# Create fresh test database from schema
echo "🔨 Creating test database with exported schema..."
sqlite3 "$TEST_DB" < "$SCHEMA_FILE"

# Insert dummy data
echo "📝 Inserting dummy test data..."
sqlite3 "$TEST_DB" <<SQL
BEGIN;

-- Insert a dummy handle (phone number)
INSERT INTO handle (id, country, service)
  VALUES ('+10005551234','US','iMessage');

-- Insert a dummy chat
INSERT INTO chat (guid, display_name, style, chat_identifier)
  VALUES ('CHAT1','Test Chat',0,'CHAT1');

-- Link chat to handle (creates one-on-one conversation)
INSERT INTO chat_handle_join (chat_id, handle_id)
  VALUES (1, 1);

-- Insert a test message at noon on 2024-01-01 (Apple epoch nanoseconds)
-- This timestamp ensures the message falls within [2024-01-01, 2024-01-02)
INSERT INTO message (
    guid, text, handle_id, date, date_read, date_delivered
) VALUES (
    'MSG1',
    'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
    1,
    (julianday('2024-01-01 12:00:00') - julianday('2001-01-01 00:00:00')) * 86400000000000,
    (julianday('2024-01-01 12:00:00') - julianday('2001-01-01 00:00:00')) * 86400000000000,
    (julianday('2024-01-01 12:00:00') - julianday('2001-01-01 00:00:00')) * 86400000000000
);

-- Join message to chat
INSERT INTO chat_message_join (chat_id, message_id)
  VALUES (1, 1);

-- Insert a second message for more realistic testing
INSERT INTO message (
    guid, text, handle_id, date, date_read, date_delivered
) VALUES (
    'MSG2',
    'Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.',
    1,
    (julianday('2024-01-01 14:30:00') - julianday('2001-01-01 00:00:00')) * 86400000000000,
    (julianday('2024-01-01 14:30:00') - julianday('2001-01-01 00:00:00')) * 86400000000000,
    (julianday('2024-01-01 14:30:00') - julianday('2001-01-01 00:00:00')) * 86400000000000
);

-- Join second message to chat
INSERT INTO chat_message_join (chat_id, message_id)
  VALUES (1, 2);

COMMIT;
SQL

echo "✅ Test database created successfully at: $TEST_DB"

# Verify the test database works with imessage-exporter
echo "🧪 Verifying test database with imessage-exporter..."
TEMP_EXPORT_DIR=$(mktemp -d)
if imessage-exporter \
    --db-path "$TEST_DB" \
    --format txt \
    --copy-method basic \
    --export-path "$TEMP_EXPORT_DIR" \
    --start-date 2024-01-01 \
    --end-date 2024-01-02 \
    --no-lazy > /dev/null 2>&1; then
    echo "✅ Test database verification successful!"
    # Check if the expected file was created
    if [[ -f "$TEMP_EXPORT_DIR/2024/01/01/+10005551234.txt" ]]; then
        echo "✅ Expected output file created: +10005551234.txt"
    else
        echo "⚠️  Expected output file not found, but export completed"
    fi
else
    echo "❌ Test database verification failed!"
    exit 1
fi

# Clean up
rm -rf "$TEMP_EXPORT_DIR"
rm -f "$SCHEMA_FILE"

echo ""
echo "🎉 Test database generation complete!"
echo "   Database: $TEST_DB"
echo ""
echo "💡 This test database contains:"
echo "   - 1 dummy contact: +10005551234"
echo "   - 1 chat conversation"
echo "   - 2 test messages on 2024-01-01"
echo ""
echo "📋 Next steps:"
echo "   - Run 'make test' to use the new test database"
echo "   - If the macOS chat.db schema changes, re-run 'make generate-test-db'"