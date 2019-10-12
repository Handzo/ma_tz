package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*
Программа читает из stdin строки, содержащие URL.
На каждый URL нужно отправить HTTP-запрос методом GET
и посчитать кол-во вхождений строки "Go" в теле ответа.
В конце работы приложение выводит на экран общее количество
найденных строк "Go" во всех переданных URL, например:
$ echo -e 'https://golang.org\nhttps://golang.org' | go run 1.go
Count for https://golang.org: 9
Count for https://golang.org: 9
Total: 18
Каждый URL должен начать обрабатываться сразу после вычитывания
и параллельно с вычитыванием следующего.
URL должны обрабатываться параллельно, но не более k=5 одновременно.
Обработчики URL не должны порождать лишних горутин, т.е. если k=5,
а обрабатываемых URL-ов всего 2, не должно создаваться 5 горутин.
Нужно обойтись без глобальных переменных и использовать только стандартную библиотеку.
*/

func main() {
	// "Go" keyword presense counter
	var total int64 = 0

	// allowed goroutines amount
	gonum := 5
	// blocking channel
	block := make(chan struct{}, gonum)

	// wait group to wait for all urls to be parsed
	wg := new(sync.WaitGroup)

	reader := bufio.NewReader(os.Stdin)

	// http client, compression disabled intentionally
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:       gonum,
			IdleConnTimeout:    10 * time.Second,
			DisableCompression: true,
		},
	}

	for {
		// block main goroutine if allowed amount of goroutines (goenum)
		// already has been executed
		block <- struct{}{}

		// read url as a string from stdin
		urlStr, err := reader.ReadString('\n')
		if err == io.EOF {
			// quit on EOF
			break
		}

		if err != nil {
			panic(err)
		}

		wg.Add(1)

		// remove trailing control characters
		urlStr = strings.TrimRight(urlStr, "\n\r")

		// start parsing gouroutine
		go func() {
			defer func() {
				wg.Done()
				// unblock main goroutine
				<-block
			}()

			url, err := url.ParseRequestURI(urlStr)
			if err != nil {
				return
			}

			// make http request
			resp, err := client.Get(url.String())
			if err != nil {
				return
			}

			// close body
			defer resp.Body.Close()

			// read all bytes from response's body
			page, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			// count "Go" keyword occurrence in the body of the response
			n := strings.Count(string(page), "Go")

			// Print out amount on certain url
			fmt.Printf("Url: %s - \"Go\" occurrence: %d\n", url.String(), n)

			// atomically add to total counter
			total = atomic.AddInt64(&total, int64(n))
		}()
	}

	wg.Wait()
	// Print out total amount
	fmt.Println("Total:", total)
}
