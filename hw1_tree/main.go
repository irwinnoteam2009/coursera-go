package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(output io.Writer, dir string, printFiles bool) error {
	return myWalker(output, dir, printFiles, "")
}

func myWalker(w io.Writer, dir string, printFiles bool, prefix string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	files = prepareFiles(files, printFiles)
	sort.Sort(FileInfos(files))
	l := len(files)
	for i, f := range files {
		fmt.Fprintln(w, showFileInfo(prefix, f, i == l-1, printFiles))

		if f.IsDir() {
			newDir := dir + string(os.PathSeparator) + f.Name()
			myWalker(w, newDir, printFiles, getNewPrefix(prefix, i == l-1))
		}
	}

	return err
}

func showFileInfo(prefix string, file os.FileInfo, isLast bool, showSize bool) string {
	pref := map[bool]string{false: "├───", true: "└───"}
	info := fmt.Sprintf("%s%s%s", prefix, pref[isLast], file.Name())
	if !file.IsDir() && showSize {
		size := file.Size()
		if size == 0 {
			info = info + " (empty)"
		} else {
			info = fmt.Sprintf("%s (%db)", info, size)
		}
	}
	return info
}

func getNewPrefix(prefix string, isLast bool) string {
	pref := map[bool]string{false: "│\t", true: "\t"}
	return fmt.Sprintf("%s%s", prefix, pref[isLast])
}

func prepareFiles(files []os.FileInfo, showFiles bool) []os.FileInfo {
	res := make([]os.FileInfo, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			res = append(res, f)
		}
		if !f.IsDir() && showFiles {
			res = append(res, f)
		}
	}
	return res
}

type FileInfos []os.FileInfo

func (a FileInfos) Len() int           { return len(a) }
func (a FileInfos) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a FileInfos) Less(i, j int) bool { return a[i].Name() < a[j].Name() }
