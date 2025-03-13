package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const workspace = "terraform-workspace"

func main() {
	fmt.Println("Starting Terraform REPL. Commands: 'upload <file>', 'delete <file>', 'exit'")
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
		if len(parts) < 2 {
			fmt.Println("Invalid command. Use 'upload <file>' or 'delete <file>'")
			continue
		}

		cmd, arg := parts[0], parts[1]
		if err := handleCmd(arg, cmd); err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}

func handleCmd(fileName string, cmd string) error {
	cmdMap := map[string]string{
		"upload": "apply",
		"delete": "destroy",
	}

	terraformCmd, exists := cmdMap[cmd]
	if !exists {
		return fmt.Errorf("Unknown command. Use 'upload <file>' or 'delete <file>'")
	}

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", fileName)
	}

	info, err := os.Stat(workspace)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Workspace directory does not exist: %s", workspace)
		} else {
			return fmt.Errorf("error checking workspace: %v", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("workspace path %s exists but is not a directory", workspace)
	}

	if err := copyFile(fileName, filepath.Join(workspace, fileName)); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	fileNameUnSuffixed := strings.TrimSuffix(fileName, ".json")
	parquetFileName := fileNameUnSuffixed + ".parquet"

	if err := runCmd(workspace, "terraform", "init", "-backend-config", fmt.Sprintf("key=state/%s.tfstate", fileNameUnSuffixed), "-reconfigure"); err != nil {
		return fmt.Errorf("terraform init failed: %v", err)
	}

	prefix := []string{}
	for key, value := range map[string]string{"file_name": fileName, "parquet_file_name": parquetFileName} {
		prefix = append(prefix, fmt.Sprintf("TF_VAR_%s=%s", key, value))
	}

	finalCmd := strings.Join(append(prefix, fmt.Sprintf("terraform %s", terraformCmd)), " ")

	if err := runCmd(workspace, finalCmd, "-auto-approve"); err != nil {
		return fmt.Errorf("terraform %s failed: %v", terraformCmd, err)
	}

	if err := os.Remove(filepath.Join(workspace, fileName)); err != nil {
		fmt.Printf("Warning: Failed to remove %s from workspace: %v\n", fileName, err)
	}
	if _, err := os.Stat(filepath.Join(workspace, parquetFileName)); !os.IsNotExist(err) {
		if err := os.Remove(filepath.Join(workspace, parquetFileName)); err != nil {
			fmt.Printf("Warning: Failed to remove %s from workspace: %v\n", parquetFileName, err)
		}
	}

	fmt.Printf("Successfully %sed %s and %s\n", cmd, fileName, parquetFileName)
	return nil
}

func runCmd(workingDir, command string, args ...string) error {
	fullCommand := append([]string{command}, args...)
	finalCmd := strings.Join(fullCommand, " ")

	cmd := exec.Command("sh", "-c", finalCmd)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
