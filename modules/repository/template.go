package repository

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"code.gitea.io/gitea/modules/options"
)

func GetGitTemplateFiles(name string) ([]string, error) {
	files, err := options.ListGitTemplateFiles(name)
	if err != nil {
		return nil, fmt.Errorf("GetFiles[%s]: %w", name, err)
	}
	return files, nil
}

func GetGitTemplate(path, name string) ([]byte, error) {
	data, err := options.GitTemplate(path, name)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("GetRepoInitFile[%s]: %w", name, err)
	}

	return data, nil
}

// CopyDir 递归拷贝一个目录下的所有文件和子目录到指定目录
func CopyGitTemplateDir(src, dest string) error {
	baseSrc := options.GetGitTemplateBaseDir(src)
	// 检查源目录是否存在
	if _, err := os.Stat(baseSrc); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", baseSrc)
	}

	// 检查目标路径是否是源路径的子目录
	base := filepath.Base(baseSrc)
	if strings.HasPrefix(filepath.Base(dest), base) {
		return fmt.Errorf("destination directory is a subdirectory of source directory")
	}

	return filepath.Walk(baseSrc, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过源目录本身的复制
		if path == baseSrc {
			return nil
		}

		relPath, err := filepath.Rel(baseSrc, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		source, err := os.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()

		// 检查目标路径是否为符号链接
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not allowed: %s", path)
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, source)
		return err
	})
}
