provider "aws" {
  profile = "default"
  region  = "us-west-1"
}

resource "aws_s3_bucket" "terraform_state" {
  bucket = "terraform-state-bucket-terra-fs"

  tags = {
    Name        = "Terraform State Bucket"
    Environment = "Shared"
  }
}

resource "aws_s3_bucket_versioning" "state_versioning" {
  bucket = aws_s3_bucket.terraform_state.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_kms_key" "state_key" {
  description             = "KMS key for Terraform state bucket"
  deletion_window_in_days = 7
}

resource "aws_s3_bucket_server_side_encryption_configuration" "state_encryption" {
  bucket = aws_s3_bucket.terraform_state.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.state_key.arn
      sse_algorithm     = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "state_security" {
  bucket = aws_s3_bucket.terraform_state.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "null_resource" "cleanup_state_bucket" {
  # Trigger only on destroy
  triggers = {
    bucket_name = aws_s3_bucket.terraform_state.bucket
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<EOT
      aws s3api list-object-versions --bucket ${self.triggers.bucket_name} --query 'Versions[].{Key:Key,VersionId:VersionId}' --output text | while read key version; do aws s3api delete-object --bucket ${self.triggers.bucket_name} --key "$key" --version-id "$version"; done || true
      aws s3api list-object-versions --bucket ${self.triggers.bucket_name} --query 'DeleteMarkers[].{Key:Key,VersionId:VersionId}' --output text | while read key version; do aws s3api delete-object --bucket ${self.triggers.bucket_name} --key "$key" --version-id "$version"; done || true
    EOT
  }

  # Ensure this runs before the bucket is deleted
  depends_on = [aws_s3_bucket.terraform_state]
}
