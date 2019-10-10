package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
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

// counter with a mutex
type counter struct {
	total    int
	totalMux sync.Mutex
}

func (c *counter) Add(n int) {
	c.totalMux.Lock()
	defer c.totalMux.Unlock()
	c.total = c.total + n
}

func main() {
	// allowed goroutines amount
	gonum := 5
	// blocking channel
	block := make(chan struct{}, gonum)

	// wait group to wait for all urls to be parsed
	wg := new(sync.WaitGroup)

	// "Go" keyword presense counter
	counter := counter{}

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
			// make http request
			resp, err := client.Get(urlStr)
			if err != nil {
				panic(err)
			}

			// close body
			defer resp.Body.Close()

			// read all bytes from response's body
			page, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}

			// count "Go" keyword occurrence in the body of the response
			n := strings.Count(string(page), "Go")

			// add to goroutine safe counter
			counter.Add(n)

			wg.Done()

			// unblock main goroutine
			<-block
		}()
	}

	wg.Wait()
	// Printout the result
	fmt.Println(counter.total)
}
