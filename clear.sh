#!/bin/bash

# Load environment variables
if [ -f .env ]; then
  source .env
else
  echo "Error: .env file not found."
  exit 1
fi

echo "Wiping generated levels from the database..."

# Execute SQL inside the running MariaDB container
docker exec -i roguelite_db mysql -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -e "TRUNCATE TABLE levels;"

echo "World cleared! Level 0 will regenerate the next time the server boots."