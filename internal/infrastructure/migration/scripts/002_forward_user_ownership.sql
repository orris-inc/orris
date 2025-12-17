-- +goose Up
ALTER TABLE forward_rules
    ADD COLUMN user_id BIGINT UNSIGNED NULL AFTER agent_id;

CREATE INDEX idx_forward_rules_user_id ON forward_rules(user_id);
CREATE INDEX idx_forward_rules_user_status ON forward_rules(user_id, status);

-- +goose Down
DROP INDEX idx_forward_rules_user_status ON forward_rules;
DROP INDEX idx_forward_rules_user_id ON forward_rules;
ALTER TABLE forward_rules DROP COLUMN user_id;
