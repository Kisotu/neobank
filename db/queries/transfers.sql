-- name: CreateTransfer :one
INSERT INTO transfers (
    from_account_id,
    to_account_id,
    amount,
    currency,
    status,
    reference_number,
    description,
    completed_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
) RETURNING *;

-- name: GetTransferByID :one
SELECT *
FROM transfers
WHERE id = $1
LIMIT 1;

-- name: GetTransferByReference :one
SELECT *
FROM transfers
WHERE reference_number = $1
LIMIT 1;

-- name: UpdateTransferStatus :exec
UPDATE transfers
SET status = $2,
    completed_at = $3
WHERE id = $1;

-- name: ListTransfersByAccount :many
SELECT *
FROM transfers
WHERE from_account_id = $1
   OR to_account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPendingTransfers :many
SELECT *
FROM transfers
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;