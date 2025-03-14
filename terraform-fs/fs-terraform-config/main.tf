terraform {
    backend "s3" {
        bucket         = "terraform-state-bucket-terra-fs"
        key            = "file.tfstate"
        region         = "us-west-1"
        profile        = "default"
        encrypt        = true
    }
}

provider "aws" {
  profile = "default"
  region  = "us-west-1"
}

data "terraform_remote_state" "root" {
  backend = "s3"
  config = {
    bucket  = "terraform-state-bucket-terra-fs"
    key     = "state/root.tfstate"
    region  = "us-west-1"
    profile = "default"
  }
}

resource "aws_s3_object" "original_file" {
  bucket       = data.terraform_remote_state.root.outputs.root_bucket_name
  key          = var.file_name
  source       = var.file_name
  content_type = "application/json"
}

resource "null_resource" "convert_to_parquet" {
  triggers = {
    original_file = aws_s3_object.original_file.id
  }

  provisioner "local-exec" {
    command = "go run convert.go ${var.file_name} ${var.parquet_file_name}"
  }

  depends_on = [aws_s3_object.original_file]
}

resource "aws_s3_object" "parquet_file" {
  bucket       = data.terraform_remote_state.root.outputs.root_bucket_name
  key          = var.parquet_file_name
  source       = var.parquet_file_name
  content_type = "application/octet-stream"

  depends_on = [null_resource.convert_to_parquet]
}
