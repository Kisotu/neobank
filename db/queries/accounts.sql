-- name: CreateAccount :one
INSERT INTO accounts (
    user_id,
    account_number,
    account_type,
    balance,
    currency,
    status
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
) RETURNING *;

-- name: GetAccountByID :one
SELECT *
FROM accounts
WHERE id = $1
LIMIT 1;

-- name: GetAccountByNumber :one
SELECT *
FROM accounts
WHERE account_number = $1
LIMIT 1;

-- name: ListAccountsByUserID :many
SELECT *
FROM accounts
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateAccountBalance :exec
UPDATE accounts
SET balance = $2,
    version = version + 1,
    updated_at = NOW()
WHERE id = $1
  AND version = $3;

-- name: UpdateAccountStatus :exec
UPDATE accounts
SET status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ListAccountsWithLock :many
SELECT *
FROM accounts
WHERE id = ANY($1::uuid[])
ORDER BY id
FOR UPDATE;