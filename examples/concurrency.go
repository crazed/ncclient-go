package main

import (
	"bytes"
	"fmt"
	"github.com/crazed/ncclient-go"
	"io"
	"os"
	"strings"
	"launchpad.net/xmlpath"
)

type WorkResult struct {
	success bool
	output  io.Reader
	client  *ncclient.Ncclient
}

func worker(id int, jobs <-chan *ncclient.Ncclient, results chan<- *WorkResult) {
	for client := range jobs {
		result := new(WorkResult)
		result.client = client
		if err := client.Connect(); err != nil {
			result.success = false
			result.output = bytes.NewBufferString(err.Error())
			results <- result
		} else {
			defer client.Close()
			client.SendHello()
			result.output = client.WriteRPC("<get-chassis-inventory/>")
			result.success = true
			results <- result
		}
	}
}

func main() {
	username := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	hosts := strings.Split(os.Getenv("HOSTS"), " ")

	jobs := make(chan *ncclient.Ncclient, len(hosts))
	results := make(chan *WorkResult, len(hosts))

	for i, host := range hosts {
		fmt.Println("Creating worker for", host)
		go worker(i, jobs, results)
	}

	for _, host := range hosts {
		client := ncclient.MakeClient(username, password, host, 22)
		jobs <- &client
	}
	close(jobs)

	description_path := xmlpath.MustCompile("//chassis/description")
	serial_number_path := xmlpath.MustCompile("//chassis/serial-number")
	for i := 0; i < len(hosts); i++ {
		result := <-results
		if result.success {
			root, _ := xmlpath.Parse(result.output)
			fmt.Println(result.client.Hostname())
			fmt.Println("-------------------------------------------")
			if description, ok := description_path.String(root); ok {
				fmt.Println("Chassis:", description)
			}
			if serial_number, ok := serial_number_path.String(root); ok {
				fmt.Println("Serial:", serial_number)
			}
			fmt.Println(" ")

		} else {
			fmt.Println(result.client.Hostname(), result.output)
		}
	}
}
