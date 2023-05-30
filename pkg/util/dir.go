package util

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func WalkDir(dir string, regex *regexp.Regexp) ([]string, error) {
	// visit 递归遍历文件夹并应用过滤器
	visit := func(files *[]string) filepath.WalkFunc {
		return func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println("Error:", err)
				return err
			}
			if !info.IsDir() && regex.MatchString(info.Name()) {
				*files = append(*files, path)
			}
			return nil
		}
	}

	var files []string
	if err := filepath.Walk(dir, visit(&files)); err != nil {
		return nil, err
	}
	return files, nil
}
