package main;

import (
	"github.com/crazed/ncclient-go"
	"fmt"
)

func main() {
	username := "root"
	password := "password"

	nc := ncclient.MakeClient(username, password, "10.200.2.1", 22)
	defer nc.Close()

	// Write a simple Hello to get going
	result := nc.Write(`<?xml version="1.0" encoding="UTF-8"?><nc:hello xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"><nc:capabilities><nc:capability>urn:ietf:params:netconf:capability:writable-running:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:rollback-on-error:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:validate:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:confirmed-commit:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:url:1.0?scheme=http,ftp,file,https,sftp</nc:capability><nc:capability>urn:ietf:params:netconf:base:1.0</nc:capability><nc:capability>urn:liberouter:params:netconf:capability:power-control:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:candidate:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:xpath:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:startup:1.0</nc:capability><nc:capability>urn:ietf:params:netconf:capability:interleave:1.0</nc:capability></nc:capabilities></nc:hello>`)

	for _, r := range result {
		fmt.Println(r)
	}

	// Request chassis inventory (juniper specific)
	result = nc.Write(`<?xml version="1.0" encoding="UTF-8"?><nc:rpc xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="urn:uuid:004e7200-7585-11e3-8919-525400625005"><nc:get-chassis-inventory/></nc:rpc>`)

	for _, r := range result {
		fmt.Println(r)
	}
	fmt.Printf("Neat i guess?\n")
}
