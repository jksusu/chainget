package helper

import (
	"log"
	"os"
	"path/filepath"
)

// ReadAbiJson 获取abi
func ReadAbiJson(fileName string) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ Failed to get working directory: %v", err)
	}
	filePath := filepath.Join(cwd, "abis", fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Failed to read ABI file %s: %v", fileName, err)
	}
	return string(content)
}
