package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/writer"
)

type Row struct {
	Index  int32  `json:"index" parquet:"name=index, type=INT32"`
	Secret string `json:"secret" parquet:"name=secret, type=BYTE_ARRAY, convertedtype=UTF8"`
}

func main() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Printf("Failed to load AWS config: %v\n", err)
		return
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)
	bucket := "not-a-bucket-2025"
	jsonKey := "not_a_file.json"
	parquetKey := "not_a_file.parquet"

	// Download the JSON file
	result, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(jsonKey),
	})
	if err != nil {
		fmt.Printf("Failed to get object: %v\n", err)
		return
	}
	defer result.Body.Close()

	// Decode JSON
	var data []Row
	if err := json.NewDecoder(result.Body).Decode(&data); err != nil {
		fmt.Printf("Failed to decode JSON: %v\n", err)
		return
	}

	// Create local parquet file
	fw, err := local.NewLocalFileWriter("temp.parquet")
	if err != nil {
		fmt.Printf("Failed to create parquet writer: %v\n", err)
		return
	}
	defer fw.Close()

	// Write to parquet
	pw, err := writer.NewParquetWriter(fw, new(Row), 4)
	if err != nil {
		fmt.Printf("Failed to create parquet writer: %v\n", err)
		return
	}

	for _, row := range data {
		if err := pw.Write(row); err != nil {
			fmt.Printf("Failed to write to parquet: %v\n", err)
			return
		}
	}

	if err = pw.WriteStop(); err != nil {
		fmt.Printf("Failed to stop parquet writer: %v\n", err)
		return
	}

	// Upload parquet file to S3
	file, err := os.Open("temp.parquet")
	if err != nil {
		fmt.Printf("Failed to open parquet file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(parquetKey),
		Body:        file,
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		fmt.Printf("Failed to upload parquet file: %v\n", err)
		return
	}

	fmt.Println("Successfully converted and uploaded file to parquet format")
	// Clean up temporary file
	os.Remove("temp.parquet")
}
