-- name: CreateTransaction :one
INSERT INTO transactions (
    account_id,
    transaction_type,
    amount,
    balance_after,
    reference_id,
    description
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
) RETURNING *;

-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = $1
LIMIT 1;

-- name: ListTransactionsByAccountID :many
SELECT *
FROM transactions
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTransactionsByAccountIDAndType :many
SELECT *
FROM transactions
WHERE account_id = $1
  AND transaction_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetTransactionsByReferenceID :many
SELECT *
FROM transactions
WHERE reference_id = $1
ORDER BY created_at ASC;

-- name: ListTransactionsByDateRange :many
SELECT *
FROM transactions
WHERE account_id = $1
  AND created_at >= $2
  AND created_at <= $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: ListTransactionsByDateRangeAndType :many
SELECT *
FROM transactions
WHERE account_id = $1
  AND created_at >= $2
  AND created_at <= $3
  AND transaction_type = $4
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;