CREATE OR REPLACE FUNCTION notify_user_changes()
RETURNS TRIGGER AS $$
DECLARE
    payload json;
BEGIN
    IF TG_OP = 'INSERT' THEN
        payload := json_build_object(
            'event_type', 'insert',
            'user', json_build_object(
                'id', NEW.id,
                'full_name', NEW.full_name,
                'email', NEW.email,
                'active', NEW.active,
                'image', NEW.image
            )
        );
    ELSIF TG_OP = 'UPDATE' THEN
        payload := json_build_object(
            'event_type', 'update',
            'user', json_build_object(
                'id', NEW.id,
                'full_name', NEW.full_name,
                'email', NEW.email,
                'active', NEW.active,
                'image', NEW.image
            )
        );
    ELSIF TG_OP = 'DELETE' THEN
        payload := json_build_object(
            'event_type', 'delete',
            'user', json_build_object(
                'id', OLD.id,
                'full_name', OLD.full_name,
                'email', OLD.email,
                'active', OLD.active,
                'image', OLD.image
            )
        );
    END IF;
    PERFORM pg_notify('user_changes', payload::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_changes_trigger
AFTER INSERT OR UPDATE OR DELETE ON gs
FOR EACH ROW
EXECUTE FUNCTION notify_user_changes();