package main

import (
	"fmt"
	"github.com/crazed/ncclient-go"
	"launchpad.net/xmlpath"
	"os"
)

func main() {
	username := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	host := os.Getenv("HOST")

	nc := ncclient.MakeClient(username, password, host, 22)
	defer nc.Close()

	// Write a simple Hello to get going
	nc.SendHello()

	// Request chassis inventory (juniper specific)
	result := nc.WriteRPC("<get-chassis-inventory/>")

	// Extract some useful information using xmlpath
	description_path := xmlpath.MustCompile("//chassis/description")
	serial_number_path := xmlpath.MustCompile("//chassis/serial-number")
	root, _ := xmlpath.Parse(result)

	if description, ok := description_path.String(root); ok {
		fmt.Println("Chassis:", description)
	}
	if serial_number, ok := serial_number_path.String(root); ok {
		fmt.Println("Serial:", serial_number)
	}
}
