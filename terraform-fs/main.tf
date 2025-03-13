terraform {
  backend "s3" {
    bucket         = "terraform-state-bucket-terra-fs"
    key            = "state/root.tfstate"
    region         = "us-west-1"
    profile        = "default"
    encrypt        = true
  }
}

provider "aws" {
  profile = "default"
  region  = "us-west-1"
}

resource "aws_s3_bucket" "root_bucket" {
  bucket = "root-bucket-terra-fs-123"

  tags = {
    Name        = "Root bucket"
    Environment = "Dev"
  }
}

resource "aws_kms_key" "root_key" {
  description             = "Encryption key for root bucket"
  deletion_window_in_days = 7
}

resource "aws_s3_bucket_server_side_encryption_configuration" "sse_bucket" {
  bucket = aws_s3_bucket.root_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.root_key.arn
      sse_algorithm     = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "bucket_security" {
  bucket = aws_s3_bucket.root_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

output "root_bucket_name" {
  value = aws_s3_bucket.root_bucket.bucket
}
