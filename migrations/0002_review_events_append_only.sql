-- review_events is append-only: reject UPDATE, DELETE and TRUNCATE.
CREATE FUNCTION review_events_reject_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'review_events is append-only (% rejected)', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;

CREATE TRIGGER review_events_no_update_delete
    BEFORE UPDATE OR DELETE ON review_events
    FOR EACH ROW EXECUTE FUNCTION review_events_reject_change();

CREATE TRIGGER review_events_no_truncate
    BEFORE TRUNCATE ON review_events
    FOR EACH STATEMENT EXECUTE FUNCTION review_events_reject_change();
