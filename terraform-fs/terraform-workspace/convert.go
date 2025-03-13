package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/apache/arrow/go/v14/parquet"
	"github.com/apache/arrow/go/v14/parquet/pqarrow"
)

func main() {
	// Expect input and output file names as arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: convert <input.json> <output.parquet>")
		os.Exit(1)
	}
	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Open and read the JSON file
	f, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Failed to open JSON file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Decode JSON as a list of arbitrary objects
	var data []map[string]interface{}
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		fmt.Printf("Failed to decode JSON: %v\n", err)
		os.Exit(1)
	}

	// Create an Arrow schema and record from the data
	schema, records, err := jsonToArrow(data)
	if err != nil {
		fmt.Printf("Failed to convert JSON to Arrow: %v\n", err)
		os.Exit(1)
	}

	// Write to Parquet
	fw, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Failed to create Parquet file: %v\n", err)
		os.Exit(1)
	}
	defer fw.Close()

	// Use default Arrow writer properties
	pw, err := pqarrow.NewFileWriter(schema, fw, parquet.NewWriterProperties(), pqarrow.ArrowWriterProperties{})
	if err != nil {
		fmt.Printf("Failed to create Parquet writer: %v\n", err)
		os.Exit(1)
	}
	defer pw.Close()

	for _, rec := range records {
		if err := pw.Write(rec); err != nil {
			fmt.Printf("Failed to write record to Parquet: %v\n", err)
			os.Exit(1)
		}
		rec.Release()
	}

	fmt.Printf("Converted %s to %s successfully\n", inputFile, outputFile)
}

// jsonToArrow converts JSON data to an Arrow schema and records
func jsonToArrow(data []map[string]interface{}) (*arrow.Schema, []arrow.Record, error) {
	if len(data) == 0 {
		// Return an empty schema and record for empty data
		fields := []arrow.Field{}
		schema := arrow.NewSchema(fields, nil)
		return schema, []arrow.Record{}, nil
	}

	// Infer schema from all rows
	fields := inferSchema(data)
	schema := arrow.NewSchema(fields, nil)

	// Build Arrow records
	pool := memory.NewGoAllocator()
	builder := array.NewRecordBuilder(pool, schema)
	defer builder.Release()

	for _, row := range data {
		for i, field := range schema.Fields() {
			switch field.Type.ID() {
			case arrow.STRING:
				b := builder.Field(i).(*array.StringBuilder)
				if val, ok := row[field.Name].(string); ok {
					b.Append(val)
				} else {
					b.AppendNull()
				}
			case arrow.FLOAT64:
				b := builder.Field(i).(*array.Float64Builder)
				if val, ok := row[field.Name].(float64); ok {
					b.Append(val)
				} else {
					b.AppendNull()
				}
			case arrow.BOOL:
				b := builder.Field(i).(*array.BooleanBuilder)
				if val, ok := row[field.Name].(bool); ok {
					b.Append(val)
				} else {
					b.AppendNull()
				}
			default:
				return nil, nil, fmt.Errorf("unsupported field type for %s", field.Name)
			}
		}
	}

	// Create a single record
	rec := builder.NewRecord()
	return schema, []arrow.Record{rec}, nil
}

// inferSchema infers an Arrow schema from JSON data
func inferSchema(data []map[string]interface{}) []arrow.Field {
	keys := make(map[string]arrow.DataType)
	for _, row := range data {
		for key, value := range row {
			if _, exists := keys[key]; !exists {
				switch value.(type) {
				case string:
					keys[key] = arrow.BinaryTypes.String
				case float64:
					keys[key] = arrow.PrimitiveTypes.Float64
				case bool:
					keys[key] = arrow.FixedWidthTypes.Boolean
				default:
					keys[key] = arrow.BinaryTypes.String // Fallback to string
				}
			}
		}
	}

	var fields []arrow.Field
	for key, typ := range keys {
		fields = append(fields, arrow.Field{Name: key, Type: typ, Nullable: true})
	}
	return fields
}
