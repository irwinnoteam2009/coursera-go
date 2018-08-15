package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mailru/easyjson"
)

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	seenBrowsers := make(map[string]bool)
	uniqueBrowsers := 0
	i := 0
	user := &User{}

	fmt.Fprintln(out, "found users:")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		readUser(scanner.Bytes(), user)

		browsers := user.Browsers

		findAndroid, uniqueAndroid := readBrowser(browsers, "Android", seenBrowsers)
		uniqueBrowsers += uniqueAndroid
		findMSIE, uniqueMSIE := readBrowser(browsers, "MSIE", seenBrowsers)
		uniqueBrowsers += uniqueMSIE

		if !(findAndroid && findMSIE) {
			i++
			continue
		}

		email := strings.Replace(user.Email, "@", " [at] ", -1)
		fmt.Fprintf(out, "[%d] %s <%s>\n", i, user.Name, email)
		i++
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Fprintln(out, "\nTotal unique browsers", len(seenBrowsers))
}

func readUser(b []byte, user *User) {
	err := easyjson.Unmarshal(b, user)
	if err != nil {
		panic(err)
	}
}

func readBrowser(browsers []string, substr string, seen map[string]bool) (bool, int) {
	unique := 0
	find := false

	for _, browser := range browsers {
		if ok := strings.Contains(browser, substr); ok {
			find = true
			if _, exists := seen[browser]; !exists {
				seen[browser] = true
				unique++
			}
		}
	}
	return find, unique
}
