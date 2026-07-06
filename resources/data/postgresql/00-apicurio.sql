-- Creates a dedicated Apicurio Registry database and user.
-- Run this against the default postgres database as a superuser.

SELECT 'CREATE USER apicurio WITH PASSWORD ''apicurio'''
    WHERE NOT EXISTS (
    SELECT 1
    FROM pg_roles
    WHERE rolname = 'apicurio'
)\gexec

SELECT 'CREATE DATABASE apicurio OWNER apicurio'
    WHERE NOT EXISTS (
    SELECT 1
    FROM pg_database
    WHERE datname = 'apicurio'
)\gexec

GRANT CONNECT ON DATABASE apicurio TO apicurio;