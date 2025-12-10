-- +goose Up

-- Fix existing nodes that have NULL short_id
-- Generate random 12-character alphanumeric strings using MySQL's random functions
UPDATE nodes SET short_id = CONCAT(
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1),
    SUBSTRING('0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz', FLOOR(1 + RAND() * 62), 1)
) WHERE short_id IS NULL OR short_id = '';

-- +goose Down

-- No rollback needed - this is a data fix migration
