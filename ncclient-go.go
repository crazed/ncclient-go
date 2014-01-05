package ncclient

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"io"
	"strconv"
)

const NETCONF_DELIM string = "]]>]]>"

type clientPassword string

func (p clientPassword) Password(user string) (string, error) {
	return string(p), nil
}

type ncclient struct {
	username string
	password string
	hostname string
	port     int

	session       *ssh.Session
	sessionStdin  io.WriteCloser
	sessionStdout io.Reader
}

func (n ncclient) Close() {
	n.session.Close()
}

func (n ncclient) Write(line string) []string {
	line = line + NETCONF_DELIM
	input := bytes.NewBufferString(line)
	b, err := n.sessionStdin.Write(input.Bytes())
	if err != nil && err != io.EOF {
		panic(err)
	}
	fmt.Printf("Wrote %d bytes: %s\n", b, input.String())

	xmlData := make([]string, 1)
	scanner := bufio.NewScanner(n.sessionStdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == NETCONF_DELIM {
			break
		}
		xmlData = append(xmlData, line)
	}
	return xmlData
}

func MakeClient(username string, password string, hostname string, port int) ncclient {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthPassword(clientPassword(password)),
		},
	}

	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", hostname, strconv.Itoa(port)), sshConfig)
	if err != nil {
		panic("Failed to dial:" + err.Error())
	}

	sshSession, err := sshClient.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}

	stdin, err := sshSession.StdinPipe()
	if err != nil {
		panic(err)
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err := sshSession.RequestSubsystem("netconf"); err != nil {
		// TODO: the command `xml-mode netconf need-trailer` can be executed
		// as a  backup if the netconf subsystem is not available, try that if we fail
		panic("Failed to make subsystem request: " + err.Error())
	}

	nc := new(ncclient)
	nc.username = username
	nc.password = password
	nc.hostname = hostname
	nc.port = port
	nc.session = sshSession
	nc.sessionStdin = stdin
	nc.sessionStdout = stdout
	return *nc
}
