import os

import boto3
import psycopg2
import urllib3
from botocore.config import Config
from psycopg2.extensions import ISOLATION_LEVEL_AUTOCOMMIT
from redis import Redis


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
    cur.execute(
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = %s AND pid <> pg_backend_pid()",
        (target_db,),
    )
    cur.execute(f'DROP DATABASE IF EXISTS "{target_db}"')
    cur.execute(f'CREATE DATABASE "{target_db}"')
conn.close()

redis_prefix = os.environ["DEFAULT_REDIS_PREFIX"]
redis_client = Redis(
    host=os.environ.get("REDIS_HOST", "localhost"),
    port=int(os.environ.get("REDIS_PORT", "6379")),
    db=int(os.environ.get("REDIS_DB_NUMBER", "0")),
    password=os.environ.get("REDIS_PASSWORD") or None,
    ssl=os.environ.get("REDIS_SSL", "").lower() == "true",
    ssl_cert_reqs="none" if os.environ.get("REDIS_SSL", "").lower() == "true" else None,
)
keys = list(redis_client.scan_iter(match=f"{redis_prefix}:*", count=1000))
if keys:
    redis_client.delete(*keys)

bucket = os.environ["S3_FILE_STORE_BUCKET_NAME"]
endpoint = os.environ.get("S3_ENDPOINT_URL") or None
access_key = os.environ.get("S3_AWS_ACCESS_KEY_ID") or None
secret_key = os.environ.get("S3_AWS_SECRET_ACCESS_KEY") or None
region = os.environ.get("AWS_REGION_NAME") or "us-east-1"
verify_ssl = os.environ.get("S3_VERIFY_SSL", "false").lower() == "true"

kwargs = {"service_name": "s3", "region_name": region}
if endpoint:
    kwargs["endpoint_url"] = endpoint
    kwargs["config"] = Config(signature_version="s3v4", s3={"addressing_style": "path"})
    if not verify_ssl:
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        kwargs["verify"] = False
if access_key and secret_key:
    kwargs["aws_access_key_id"] = access_key
    kwargs["aws_secret_access_key"] = secret_key

s3_client = boto3.client(**kwargs)
paginator = s3_client.get_paginator("list_objects_v2")
for page in paginator.paginate(Bucket=bucket):
    objects = [{"Key": item["Key"]} for item in page.get("Contents", [])]
    if objects:
        s3_client.delete_objects(Bucket=bucket, Delete={"Objects": objects})
