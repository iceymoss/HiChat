-- Apply after deploying group history support.
CREATE INDEX idx_messages_group_history
    ON messages (type, target_id, id);
