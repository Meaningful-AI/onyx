import os

import psycopg2
from psycopg2.extensions import ISOLATION_LEVEL_AUTOCOMMIT


host = os.environ.get("POSTGRES_HOST", "localhost")
port = os.environ.get("POSTGRES_PORT", "5432")
user = os.environ.get("POSTGRES_USER", "postgres")
password = os.environ.get("POSTGRES_PASSWORD", "password")
target_db = os.environ["POSTGRES_DB"]
admin_db = os.environ.get("AGENT_LAB_POSTGRES_ADMIN_DB", "postgres")

conn = psycopg2.connect(
    host=host, port=port, user=user, password=password, dbname=admin_db
)
conn.set_isolation_level(ISOLATION_LEVEL_AUTOCOMMIT)
with conn.cursor() as cur:
    cur.execute("SELECT 1 FROM pg_database WHERE datname = %s", (target_db,))
    if cur.fetchone() is None:
        cur.execute(f'CREATE DATABASE "{target_db}"')
conn.close()
print(target_db)
