-- Migration: Fix table ownership after a pg_dump restore (DP-601)
-- Date: 2026-07-11
-- Author: Claude
--
-- SYMPTOM
--   The server logs on startup and then runs without persistence:
--       WARN Database connection failed, continuing without persistence
--            error="create tables: pq: must be owner of table players (42501)"
--
-- ROOT CAUSE
--   This is NOT a code bug. createTables() (pkg/db/player.go) issues idempotent
--   migrations like `ALTER TABLE players ADD COLUMN IF NOT EXISTS ...`. PostgreSQL
--   checks table ownership BEFORE the IF NOT EXISTS short-circuit, so ALTER TABLE
--   fails with 42501 when the table is owned by someone other than the connecting
--   role. After the CT 120 restore, tables were recreated by the `postgres`
--   superuser (the role that ran the pg_dump load) instead of `darkpawns`.
--
-- FIX
--   Reassign ownership of every public-schema table, sequence, and view to the
--   `darkpawns` application role. Idempotent — safe to re-run after any restore.
--
-- HOW TO RUN (on CT 120, as a superuser, against the darkpawns database)
--   psql -U postgres -d darkpawns -f scripts/migrations/002_fix_table_ownership.sql
--
-- VERIFY
--   SELECT tablename, tableowner FROM pg_tables WHERE schemaname = 'public';
--   -- every row should read 'darkpawns'
--   Then restart the server; it should log "Database connected." instead of the
--   "continuing without persistence" warning.
--
-- ALTERNATIVE (one-liner, same effect, run as superuser in the darkpawns DB):
--   REASSIGN OWNED BY postgres TO darkpawns;

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO darkpawns', r.tablename);
    END LOOP;

    FOR r IN SELECT sequencename FROM pg_sequences WHERE schemaname = 'public' LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO darkpawns', r.sequencename);
    END LOOP;

    FOR r IN SELECT viewname FROM pg_views WHERE schemaname = 'public' LOOP
        EXECUTE format('ALTER VIEW public.%I OWNER TO darkpawns', r.viewname);
    END LOOP;
END $$;
