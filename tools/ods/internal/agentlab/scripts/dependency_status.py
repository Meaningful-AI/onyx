import json
import os

import boto3
import psycopg2
import urllib3
from botocore.config import Config
from botocore.exceptions import ClientError
from redis import Redis


db_name = os.environ["POSTGRES_DB"]
host = os.environ.get("POSTGRES_HOST", "localhost")
port = os.environ.get("POSTGRES_PORT", "5432")
user = os.environ.get("POSTGRES_USER", "postgres")
password = os.environ.get("POSTGRES_PASSWORD", "password")

conn = psycopg2.connect(
    host=host, port=port, user=user, password=password, dbname=db_name
)
with conn.cursor() as cur:
    cur.execute(
        "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'"
    )
    table_count = int(cur.fetchone()[0])
conn.close()

redis_prefix = os.environ["DEFAULT_REDIS_PREFIX"]
bucket = os.environ["S3_FILE_STORE_BUCKET_NAME"]

redis_client = Redis(
    host=os.environ.get("REDIS_HOST", "localhost"),
    port=int(os.environ.get("REDIS_PORT", "6379")),
    db=int(os.environ.get("REDIS_DB_NUMBER", "0")),
    password=os.environ.get("REDIS_PASSWORD") or None,
    ssl=os.environ.get("REDIS_SSL", "").lower() == "true",
    ssl_cert_reqs="none" if os.environ.get("REDIS_SSL", "").lower() == "true" else None,
)
redis_key_count = 0
for _ in redis_client.scan_iter(match=f"{redis_prefix}:*", count=1000):
    redis_key_count += 1

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
bucket_ready = True
bucket_object_count = 0
try:
    s3_client.head_bucket(Bucket=bucket)
    paginator = s3_client.get_paginator("list_objects_v2")
    for page in paginator.paginate(Bucket=bucket):
        bucket_object_count += len(page.get("Contents", []))
except ClientError:
    bucket_ready = False

print(
    json.dumps(
        {
            "mode": os.environ["AGENT_LAB_DEPENDENCY_MODE"],
            "namespace": os.environ.get("AGENT_LAB_NAMESPACE", ""),
            "postgres_database": db_name,
            "postgres_ready": True,
            "postgres_table_count": table_count,
            "redis_prefix": redis_prefix,
            "redis_ready": True,
            "redis_key_count": redis_key_count,
            "file_store_bucket": bucket,
            "file_store_ready": bucket_ready,
            "file_store_object_count": bucket_object_count,
            "search_infra_mode": os.environ.get(
                "AGENT_LAB_SEARCH_INFRA_MODE", "shared"
            ),
        }
    )
)
