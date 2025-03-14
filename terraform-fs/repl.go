package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const config_dir = "fs-terraform-config"

func checkConfigDir() error {
	info, err := os.Stat(config_dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Terraform config directory does not exist: %s", config_dir)
		} else {
			return fmt.Errorf("error checking config directory: %v", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("config directory path %s exists but is not a directory", config_dir)
	}
	return nil
}

func printUsage() {
	fmt.Println("\nUsage:")
	fmt.Println("  upload <file>      - Upload a JSON file and process it.")
	fmt.Println("  delete <file>      - Delete a specific uploaded file.")
	fmt.Println("  delete all         - Delete all uploaded files.")
	fmt.Println("  list               - List all uploaded files.")
	fmt.Println("  exit               - Exit the REPL.")
}

func main() {
	fmt.Println("Starting Terraform FS REPL")
	printUsage()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "exit" {
			fmt.Println("Exiting REPL...")
			os.Exit(0)
		}

		parts := strings.Fields(input)

		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		switch cmd {
		case "list":
			if len(parts) != 1 {
				fmt.Println("Usage: list")
				continue
			}
			if err := handleList(); err != nil {
				fmt.Printf("%v\n", err)
			}
		case "upload":
			if len(parts) != 2 {
				fmt.Println("Usage: upload <file>")
				continue
			}
			arg := parts[1]
			if err := handleUpload(arg); err != nil {
				fmt.Printf("%v\n", err)
			}
		case "delete":
			if len(parts) != 2 {
				fmt.Println("Usage: delete <file>|all")
				continue
			}
			arg := parts[1]
			if arg == "all" {
				if err := handleDeleteAll(); err != nil {
					fmt.Printf("%v\n", err)
				}
			} else {
				if err := handleDelete(arg); err != nil {
					fmt.Printf("%v\n", err)
				}
			}
		case "exit":
			fmt.Println("Exiting REPL...")
			os.Exit(0)
		default:
			fmt.Println("Unknown command.")
			printUsage()
		}
	}
}

func handleList() error {
	if err := checkConfigDir(); err != nil {
		return err
	}

	if _, err := runCmd(config_dir, "terraform", "init"); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	output, err := runCmd(config_dir, "terraform", "workspace", "list")
	if err != nil {
		return fmt.Errorf("failed to list terraform workspaces: %v", err)
	}

	lines := strings.Split(output, "\n")
	var activeWorkspace string
	var workspaceList []string
	for _, line := range lines {
		workspaceName := strings.TrimSpace(strings.ReplaceAll(line, "*", ""))
		if workspaceName == "" {
			continue
		}
		workspaceList = append(workspaceList, workspaceName)

		if strings.Contains(line, "*") {
			activeWorkspace = workspaceName
		}
	}

	if len(workspaceList) == 0 {
		fmt.Println("No files found")
	} else {
		fmt.Println("Available files:")
		for _, ws := range workspaceList {
			if ws == "default" {
				continue
			}
			if ws == activeWorkspace {
				fmt.Printf("- %s.json *\n", ws)
				fmt.Printf("- %s.parquet *\n", ws)
			} else {
				fmt.Printf("- %s.json\n", ws)
				fmt.Printf("- %s.parquet\n", ws)
			}
		}
	}

	return nil
}

func handleUpload(fileName string) error {
	if err := checkConfigDir(); err != nil {
		return err
	}

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", fileName)
	}

	if err := copyFile(fileName, filepath.Join(config_dir, fileName)); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	fileNameUnSuffixed := strings.TrimSuffix(fileName, ".json")
	parquetFileName := fileNameUnSuffixed + ".parquet"

	if _, err := runCmd(config_dir, "terraform", "init"); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	output, err := runCmd(config_dir, "terraform", "workspace", "list")
	if err != nil {
		return fmt.Errorf("failed to list terraform workspaces: %v", err)
	}

	workspaces := make(map[string]bool)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		workspaceName := strings.TrimSpace(strings.ReplaceAll(line, "*", ""))
		if workspaceName == "" {
			continue
		}
		workspaces[workspaceName] = true
	}

	if workspaces[fileNameUnSuffixed] {
		fmt.Printf("Warning: file %s already exists, rewriting", fileNameUnSuffixed)
		if _, err := runCmd(config_dir, "terraform", "workspace", "select", fileNameUnSuffixed); err != nil {
			return fmt.Errorf("failed to switch to terraform workspace %s: %v", fileNameUnSuffixed, err)
		}
	} else {
		if _, err := runCmd(config_dir, "terraform", "workspace", "new", fileNameUnSuffixed); err != nil {
			return fmt.Errorf("failed to create terraform workspace %s: %v", fileNameUnSuffixed, err)
		}
		workspaces[fileNameUnSuffixed] = true
	}

	prefix := []string{}
	for key, value := range map[string]string{"file_name": fileName, "parquet_file_name": parquetFileName} {
		prefix = append(prefix, fmt.Sprintf("TF_VAR_%s=%s", key, value))
	}

	finalCmd := strings.Join(append(prefix, "terraform apply"), " ")

	if _, err := runCmd(config_dir, finalCmd, "-auto-approve"); err != nil {
		return fmt.Errorf("terraform apply failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(config_dir, fileName)); !os.IsNotExist(err) {
		if err := os.Remove(filepath.Join(config_dir, fileName)); err != nil {
			fmt.Printf("Warning: Failed to remove %s from config directory: %v\n", fileName, err)
		}
	}
	if _, err := os.Stat(filepath.Join(config_dir, parquetFileName)); !os.IsNotExist(err) {
		if err := os.Remove(filepath.Join(config_dir, parquetFileName)); err != nil {
			fmt.Printf("Warning: Failed to remove %s from config directory: %v\n", parquetFileName, err)
		}
	}

	fmt.Printf("Successfully created %s and %s\n", fileName, parquetFileName)
	return nil
}

func handleDelete(fileName string) error {
	if err := checkConfigDir(); err != nil {
		return err
	}

	fileNameUnSuffixed := strings.TrimSuffix(fileName, ".json")
	parquetFileName := fileNameUnSuffixed + ".parquet"

	if _, err := runCmd(config_dir, "terraform", "init"); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	output, err := runCmd(config_dir, "terraform", "workspace", "list")
	if err != nil {
		return fmt.Errorf("failed to list terraform workspaces: %v", err)
	}

	workspaces := make(map[string]bool)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		workspaceName := strings.TrimSpace(strings.ReplaceAll(line, "*", "")) // Remove '*' if active workspace
		if workspaceName == "" {
			continue
		}
		workspaces[workspaceName] = true
	}

	if workspaces[fileNameUnSuffixed] {
		if _, err := runCmd(config_dir, "terraform", "workspace", "select", fileNameUnSuffixed); err != nil {
			return fmt.Errorf("failed to switch to terraform workspace %s: %v", fileNameUnSuffixed, err)
		}
	} else {
		return fmt.Errorf("terraform workspace for %s not found", fileNameUnSuffixed)
	}

	prefix := []string{}
	for key, value := range map[string]string{"file_name": fileName, "parquet_file_name": parquetFileName} {
		prefix = append(prefix, fmt.Sprintf("TF_VAR_%s=%s", key, value))
	}

	finalCmd := strings.Join(append(prefix, "terraform destroy"), " ")

	if _, err := runCmd(config_dir, finalCmd, "-auto-approve"); err != nil {
		return fmt.Errorf("terraform destroy failed: %v", err)
	}

	if _, err := runCmd(config_dir, "terraform", "workspace", "select", "default"); err != nil {
		return fmt.Errorf("failed to switch to default terraform workspace: %v", err)
	}
	if _, err := runCmd(config_dir, "terraform", "workspace", "delete", fileNameUnSuffixed); err != nil {
		return fmt.Errorf("failed to delete terraform workspace %s: %v", fileNameUnSuffixed, err)
	}

	fmt.Printf("Successfully deleted %s and %s\n", fileName, parquetFileName)
	return nil
}

func handleDeleteAll() error {
	if err := checkConfigDir(); err != nil {
		return err
	}

	if _, err := runCmd(config_dir, "terraform", "init"); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	output, err := runCmd(config_dir, "terraform", "workspace", "list")
	if err != nil {
		return fmt.Errorf("failed to list terraform workspaces: %v", err)
	}

	var workspaces []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		workspaceName := strings.TrimSpace(strings.ReplaceAll(line, "*", ""))
		if workspaceName != "" && workspaceName != "default" {
			workspaces = append(workspaces, workspaceName)
		}
	}

	if len(workspaces) == 0 {
		fmt.Println("No Terraform workspaces to delete.")
		return nil
	}

	for _, ws := range workspaces {
		if _, err := runCmd(config_dir, "terraform", "workspace", "select", ws); err != nil {
			return fmt.Errorf("failed to switch to terraform workspace %s: %v", ws, err)
		}

		prefix := []string{}
		for key, value := range map[string]string{"file_name": fmt.Sprintf("%s.json", ws), "parquet_file_name": fmt.Sprintf("%s.parquet", ws)} {
			prefix = append(prefix, fmt.Sprintf("TF_VAR_%s=%s", key, value))
		}

		finalCmd := strings.Join(append(prefix, "terraform destroy"), " ")

		if _, err := runCmd(config_dir, finalCmd, "-auto-approve"); err != nil {
			return fmt.Errorf("terraform destroy failed: %v", err)
		}

		if _, err := runCmd(config_dir, "terraform", "workspace", "select", "default"); err != nil {
			return fmt.Errorf("failed to switch to default terraform workspace: %v", err)
		}
		if _, err := runCmd(config_dir, "terraform", "workspace", "delete", ws); err != nil {
			return fmt.Errorf("failed to delete terraform workspace %s: %v", ws, err)
		}

		fmt.Printf("Successfully deleted %s.json and %s.parquet\n", ws, ws)
	}

	fmt.Printf("Successfully deleted all files\n")
	return nil
}

func runCmd(workingDir, command string, args ...string) (string, error) {
	fullCommand := append([]string{command}, args...)
	finalCmd := strings.Join(fullCommand, " ")

	var outputBuffer bytes.Buffer

	cmd := exec.Command("sh", "-c", finalCmd)
	cmd.Dir = workingDir

	// If we want terraform outputs additionally in the REPL, we could also do
	// cmd.Stdout = io.MultiWriter(os.Stdout, &outputBuffer)
	cmd.Stdout = &outputBuffer

	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return outputBuffer.String(), err
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}
