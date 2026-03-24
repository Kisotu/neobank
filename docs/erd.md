# Banking App ERD

## Entities

### users
- `id` (PK, UUID)
- `email` (unique)
- `password_hash`
- `full_name`
- `deleted_at` (soft delete marker)
- `created_at`
- `updated_at`

### accounts
- `id` (PK, UUID)
- `user_id` (FK -> users.id)
- `account_number` (unique)
- `account_type` (`checking` | `savings`)
- `balance` (DECIMAL(19,4))
- `currency` (ISO-4217 code)
- `status` (`active` | `frozen` | `closed`)
- `version` (optimistic lock version)
- `created_at`
- `updated_at`

### transactions
- `id` (PK, UUID)
- `account_id` (FK -> accounts.id)
- `transaction_type` (`deposit` | `withdrawal` | `transfer_in` | `transfer_out`)
- `amount` (DECIMAL(19,4))
- `balance_after` (DECIMAL(19,4))
- `reference_id` (optional UUID to correlate records)
- `description`
- `created_at`

### transfers
- `id` (PK, UUID)
- `from_account_id` (FK -> accounts.id)
- `to_account_id` (FK -> accounts.id)
- `amount` (DECIMAL(19,4))
- `currency`
- `status` (`pending` | `completed` | `failed` | `reversed`)
- `reference_number` (unique)
- `description`
- `created_at`
- `completed_at`

## Relationships

- One `user` has many `accounts`.
- One `account` has many `transactions`.
- One `transfer` references two `accounts`: source and destination.
- `transactions.reference_id` can be used to group two transfer-side entries (`transfer_out` + `transfer_in`) around the same business event.

## Constraints and Concurrency

- `accounts.version` supports optimistic locking in update queries.
- Transfer operations should use row locking (`FOR UPDATE`) on involved accounts in stable sorted order.
- Money values are stored in `DECIMAL(19,4)`.