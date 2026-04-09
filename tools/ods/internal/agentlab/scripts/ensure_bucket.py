import os

import boto3
import urllib3
from botocore.config import Config
from botocore.exceptions import ClientError


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

client = boto3.client(**kwargs)
try:
    client.head_bucket(Bucket=bucket)
except ClientError as exc:
    status = exc.response.get("ResponseMetadata", {}).get("HTTPStatusCode")
    if status not in (403, 404):
        raise
    if endpoint or region == "us-east-1":
        client.create_bucket(Bucket=bucket)
    else:
        client.create_bucket(
            Bucket=bucket, CreateBucketConfiguration={"LocationConstraint": region}
        )
print(bucket)
