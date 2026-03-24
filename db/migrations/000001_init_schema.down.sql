DROP TRIGGER IF EXISTS trigger_accounts_updated_at ON accounts;
DROP TRIGGER IF EXISTS trigger_users_updated_at ON users;

DROP FUNCTION IF EXISTS set_updated_at();
DROP FUNCTION IF EXISTS generate_account_number();

DROP TABLE IF EXISTS transfers;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS "uuid-ossp";