BEGIN;
LOCK TABLE albums IN ACCESS EXCLUSIVE MODE;
SELECT pg_sleep(:'hold_seconds'::double precision);
ROLLBACK;
