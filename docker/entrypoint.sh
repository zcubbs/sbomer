#!/bin/sh

# Optional: Run database migrations
go run scripts/migrate.go || echo "Migrations failed or skipped"

# Start both services in the background
/app/bin/sbomer &
/app/bin/fetcher &

# Wait for either process to exit to keep the container alive
wait -n

# Exit with the status code of the first process that exits
exit $?
