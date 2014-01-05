package main

import (
	"bytes"
	"fmt"
	"github.com/crazed/ncclient-go"
	"launchpad.net/xmlpath"
	"os"
)

func main() {
	username := os.Getenv("USER")
	password := os.Getenv("PASSWORD")

	nc := ncclient.MakeClient(username, password, "10.200.2.1", 22)
	defer nc.Close()

	// Write a simple Hello to get going
	nc.SendHello()

	// Request chassis inventory (juniper specific)
	result := nc.WriteRPC("<get-chassis-inventory/>")

	// Extract some useful information using xmlpath
	description_path := xmlpath.MustCompile("//chassis/description")
	serial_number_path := xmlpath.MustCompile("//chassis/serial-number")
	// If WriteRPC returned io.Reader, we wouldn't need to create a buffer string
	b := bytes.NewBufferString(result)
	root, _ := xmlpath.Parse(b)

	if description, ok := description_path.String(root); ok {
		fmt.Println("Chassis:", description)
	}
	if serial_number, ok := serial_number_path.String(root); ok {
		fmt.Println("Serial:", serial_number)
	}
}
