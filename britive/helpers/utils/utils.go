package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func ExpandStringList(v interface{}) []string {
	list := v.([]interface{})
	result := make([]string, len(list))
	for i, val := range list {
		result[i] = val.(string)
	}
	return result
}

func HashFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
