-- name: CreateOperationHistory :one
INSERT INTO operation_history (
    operation,
    project_id,
    description
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetOperationHistoryByProjectID :many
SELECT * FROM operation_history
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: GetLatestOperations :many
SELECT * FROM operation_history
ORDER BY created_at DESC
LIMIT $1;

-- name: GetOperationsByType :many
SELECT * FROM operation_history
WHERE operation = $1
ORDER BY created_at DESC;

-- name: DeleteOldOperations :exec
DELETE FROM operation_history
WHERE created_at < NOW() - INTERVAL '$1 days';
