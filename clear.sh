#!/bin/bash

# Load environment variables
if [ -f .env ]; then
  source .env
else
  echo "Error: .env file not found."
  exit 1
fi

echo "========================================================"
echo " WARNING: TOTAL DATABASE WIPE INITIATED"
echo "========================================================"
echo "This will delete ALL players, passwords, stats, and maps."
read -p "Are you absolutely sure you want to proceed? (y/n) " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted. Your database is safe."
    exit 1
fi

echo "1. Dropping existing database..."
# Execute the DROP command inside the container
docker exec -i roguelite_db mysql -u"$DB_USER" -p"$DB_PASSWORD" -e "DROP DATABASE IF EXISTS $DB_NAME;"

echo "2. Rebuilding database and tables from schema.sql..."
# Pipe the schema.sql file directly into the MySQL container
cat schema.sql | docker exec -i roguelite_db mysql -u"$DB_USER" -p"$DB_PASSWORD"

echo "========================================================"
echo "Success! VeggieMUD is completely wiped and rebuilt."
echo "Level 0 will regenerate the next time you start the server."