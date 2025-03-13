# Terraform FS Project 🚀

This project provides a **Terraform REPL (Read-Eval-Print Loop)** that allows users to manage AWS infrastructure (a file system in this case) dynamically using Terraform. It enables:
- **Uploading files** and managing their lifecycle via Terraform.
- **Automatic JSON-to-Parquet conversion** before storage.
- **State management with S3 backend**.

---

## 🚀 Features
- **Interactive REPL**: Run Terraform commands dynamically.
- **Automated state management**: Uses **S3 backend** to store Terraform state files.
- **Secure storage**: Server-side Encryption using **AWS KMS**.
- **JSON to Parquet conversion**: Automatically converts `.json` files to `.parquet` before uploading.

---

## 📦 Project Structure

```
terraform-fs-root
├── README.md
├── .gitignore
├── bootstrap
│   ├── main.tf
├── terraform-fs
│   └── infra
│       ├── main.tf
│   ├── mtcars.json
│   ├── repl.go
│   ├── sample.json
│   └── fs-workspace
│       ├── convert.go
│       ├── go.mod
│       ├── go.sum
│       ├── main.tf
│       └── variables.tf
└── tutorial-root
    ├── bootstrap
    │   ├── main.tf
    └── tutorial
        ├── go.mod
        ├── go.sum
        ├── main.go
        └── main.tf
```

## 🔧 Prerequisites
Before using this Terraform REPL, ensure you have the following installed:
- [Terraform](https://developer.hashicorp.com/terraform/downloads) (≥ v1.0)
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
- [Go](https://go.dev/doc/install) (≥ v1.22)
- AWS credentials configured (`aws configure`)

## 🛠️ Setup Instructions
This only needs to be done the first time
1. Clone the repository
2. Navigate to `bootstrap`
3. Run `terraform init` to initialize the Terraform working directory
4. Run `terraform apply -auto-approve`

## Usage
1. Navigate to `terraform-fs/infra`
2. Run `terraform init` and `terraform apply -auto-approve`
3. Navigate to `terraform-fs` and run `go run repl.go` to start the Terraform REPL
4. Once the REPL starts, you can use commands like:
```
> upload sample.json
> delete sample.json
> exit
```
