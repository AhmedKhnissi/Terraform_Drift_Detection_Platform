terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Credentials are resolved from the AWS default credential chain:
# ~/.aws/credentials (profile = "default"), which is already configured on this
# machine. Region is pinned to match the drift-detection platform config.
provider "aws" {
  region  = "us-east-1"
  profile = "default"
}

# Bucket name is globally unique thanks to the AWS account id suffix
# (796973496507). The drift detector matches this resource by its id (the bucket
# name) and compares the tags below against the live AWS resource.
resource "aws_s3_bucket" "drift_demo" {
  bucket = "drift-demo-796973496507"

  tags = {
    Name        = "drift-demo-796973496507"
    Environment = "demo"
    ManagedBy   = "terraform"
    Project     = "drift-detection-platform"
  }
}

# Versioning is configured via the dedicated resource (AWS provider v4+).
resource "aws_s3_bucket_versioning" "drift_demo" {
  bucket = aws_s3_bucket.drift_demo.id

  versioning_configuration {
    status = "Enabled"
  }
}

# SSE with S3-managed keys (AES256).
resource "aws_s3_bucket_server_side_encryption_configuration" "drift_demo" {
  bucket = aws_s3_bucket.drift_demo.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Explicitly block all public access (defense in depth; S3 already defaults to
# blocking public access at the account level, but this makes intent explicit).
resource "aws_s3_bucket_public_access_block" "drift_demo" {
  bucket                  = aws_s3_bucket.drift_demo.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
