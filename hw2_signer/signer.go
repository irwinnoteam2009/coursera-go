package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

type task struct {
	id  int
	crc string
}

type taskArr []task

func (a taskArr) Less(i, j int) bool { return a[i].id < a[j].id }
func (a taskArr) Len() int           { return len(a) }
func (a taskArr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func addTask(ch chan<- task, id int, data string) {
	go func() {
		ch <- task{id, DataSignerCrc32(data)}
	}()
}

// SingleHash returns crc32(data)+"~"+crc32(md5(data))
func SingleHash(in, out chan interface{}) {
	result := make(chan task)

	i := 0
	for v := range in {
		data := strconv.Itoa(v.(int))
		md5 := DataSignerMd5(data)
		fmt.Println(data, "SingleHash data", data)
		fmt.Println(data, "SingleHash md5(data)", md5)

		addTask(result, 0, data)
		addTask(result, 1, md5)
		i++
	}

	for j := 0; j < i; j++ {
		tasks := make(taskArr, 2)
		tasks[0] = <-result
		tasks[1] = <-result

		sort.Sort(tasks)
		crc32 := tasks[0].crc
		crc32md5 := tasks[1].crc
		hash := crc32 + "~" + crc32md5

		fmt.Println("SingleHash crc32(md5(data))", crc32md5)
		fmt.Println("SingleHash crc32(data)", crc32)
		fmt.Println("SingleHash result", hash)

		out <- hash
	}
}

// MultiHash returns crc32(th+data))
func MultiHash(in, out chan interface{}) {
	result := make(chan task)
	c := 0
	for d := range in {
		data := d.(string)
		for i := 0; i <= 5; i++ {
			addTask(result, i, strconv.Itoa(i)+data)
		}
		c++
	}

	for j := 0; j < c; j++ {
		b := new(bytes.Buffer)
		tasks := make(taskArr, 6)
		for i := 0; i <= 5; i++ {
			tasks[i] = <-result
		}
		sort.Sort(tasks)
		for i := 0; i <= 5; i++ {
			str := tasks[i].crc
			fmt.Println("MultiHash: crc32(th+step1)", j, str)
			b.WriteString(str)
		}

		hash := b.String()
		fmt.Println("MutliHash result:", hash)
		out <- hash
	}
}

// CombineResults returns sorted resuls in one string with "_" delimiter
func CombineResults(in, out chan interface{}) {
	arr := make([]string, 0, 100)
	for d := range in {
		data := d.(string)
		arr = append(arr, data)
	}

	sort.Strings(arr)

	b := new(bytes.Buffer)
	for i, v := range arr {
		if i != 0 {
			b.WriteString("_")
		}
		b.WriteString(v)
	}

	hash := b.String()
	fmt.Println("CombineResults ", hash)
	out <- hash
}

// ExecutePipeline execute all jobs
func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})
	wg := new(sync.WaitGroup)

	for i, v := range jobs {
		out := make(chan interface{})
		wg.Add(1)
		go func(index int, job job, in chan interface{}, out chan interface{}) {
			defer func() {
				close(out)
				wg.Done()
			}()

			job(in, out)

		}(i, v, in, out)
		in = out
	}
	wg.Wait()
}

func main() {

}
