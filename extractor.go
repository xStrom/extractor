// Copyright 2020 Kaur Kuut
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const workDir = "./work/"
const outDir = "./gravatar/"

func main() {
	dealWithFiles(workDir)
}

func dealWithFiles(dirName string) {
	// Get the file list for this directory
	fileInfos := getFileList(dirName)

	for j := range fileInfos {
		name := fileInfos[j].Name()
		fullName := filepath.Join(dirName, name)
		isDir := fileInfos[j].IsDir()

		if isDir {
			dealWithFiles(fullName)
		} else {
			if strings.HasSuffix(name, ".html") {
				//fmt.Printf("Proccessing: %v\n", name)
				processFile(fullName)
			}
		}
	}
}

func getFileList(dirName string) []os.FileInfo {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		fmt.Printf("ReadDir failed: %v\n", err)
		panic("")
	}
	return files
}

var gravatars = map[string]string{}

func processFile(fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	changes := false
	re := regexp.MustCompile(`"https://www.gravatar.com/avatar/(.{32})[^"]*"`)
	for _, match := range re.FindAllSubmatch(data, -1) {
		changes = true
		ext := dealWithGravatar(string(match[1]))
		fileName := append(append(append([]byte(`"/img/gravatar/`), match[1]...), []byte(ext)...), '"')
		data = bytes.Replace(data, match[0], fileName, -1)
	}
	if changes {
		ioutil.WriteFile(fileName, data, 0644)
	}
	return nil
}

func dealWithGravatar(hash string) string {
	if ext, ok := gravatars[hash]; ok {
		return ext
	}

	// Download it
	fmt.Printf("Downloading: %v\n", hash)
	ext, err := download(fmt.Sprintf("https://www.gravatar.com/avatar/%v?s=50&d=identicon&r=pg", hash), filepath.Join(outDir, hash))
	if err != nil {
		panic(fmt.Sprintf("Failed to download: %v", err))
	}

	gravatars[hash] = ext
	return ext
}

var jpgHeader = []byte{0xFF, 0xD8, 0xFF}
var pngHeader = []byte{0x89, 0x50, 0x4E}

func download(url string, path string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	header := make([]byte, 3)
	count, err := resp.Body.Read(header)
	if count != len(header) {
		panic(fmt.Sprintf("Expected to read %v bytes but got %v", len(header), count))
	}
	if err != nil {
		return "", err
	}

	extension := ""
	if bytes.Equal(header, jpgHeader) {
		extension = ".jpg"
	} else if bytes.Equal(header, pngHeader) {
		extension = ".png"
	} else {
		fmt.Printf("Unrecognized image header: %+v", header)
	}

	out, err := os.Create(path + extension)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err = out.Write(header); err != nil {
		return "", err
	}
	_, err = io.Copy(out, resp.Body)
	return extension, err
}
