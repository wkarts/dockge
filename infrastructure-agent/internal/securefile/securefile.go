package securefile

import (
    "os"
    "path/filepath"
    "strings"
)

func Read(path string) (string, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(b)), nil
}

func Write(path, value string) error {
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return err
    }
    return os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), 0600)
}
