//go:build !windows

package atomicfile

import (
    "os"
    "path/filepath"
)

func Replace(tempPath, targetPath string) error {
    if err := os.Rename(tempPath, targetPath); err != nil {
        return err
    }
    // Persist the directory entry when the filesystem supports fsync on dirs.
    dir, err := os.Open(filepath.Dir(targetPath))
    if err != nil {
        return nil
    }
    defer dir.Close()
    _ = dir.Sync()
    return nil
}
