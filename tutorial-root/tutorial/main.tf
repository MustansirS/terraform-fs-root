terraform {
  backend "s3" {
    bucket         = "terraform-state-bucket-tutorial-script"
    key            = "state/terraform.tfstate"
    region         = "us-west-1"
    profile        = "default"
    encrypt        = true
  }
}

provider "aws" {
  profile = "default"
  region  = "us-west-1"
}

resource "aws_s3_bucket" "not_a_bucket" {
  bucket = "not-a-bucket-2025"

  tags = {
    Name        = "Not a bucket"
    Environment = "Dev"
  }
}

resource "aws_kms_key" "not_a_key" {
  description             = "Encryption key for Not a bucket"
  deletion_window_in_days = 7
}

resource "aws_s3_bucket_server_side_encryption_configuration" "sse_bucket" {
  bucket = aws_s3_bucket.not_a_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.not_a_key.arn
      sse_algorithm     = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "bucket_security" {
  bucket = aws_s3_bucket.not_a_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_object" "upload_json" {
  bucket       = aws_s3_bucket.not_a_bucket.id
  key          = "not_a_file.json"
  content      = "[{ \"index\": 1, \"secret\": \"5a41141b12465608cf3b44db0b9e488e\" }]"
  content_type = "application/json"
}
