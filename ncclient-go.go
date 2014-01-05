package ncclient

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const NETCONF_DELIM string = "]]>]]>"
const NETCONF_HELLO string = `
<?xml version="1.0" encoding="UTF-8"?>
<nc:hello xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0">
	<nc:capabilities>
		<nc:capability>urn:ietf:params:netconf:capability:writable-running:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:rollback-on-error:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:validate:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:confirmed-commit:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:url:1.0?scheme=http,ftp,file,https,sftp</nc:capability>
		<nc:capability>urn:ietf:params:netconf:base:1.0</nc:capability>
		<nc:capability>urn:liberouter:params:netconf:capability:power-control:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:candidate:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:xpath:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:startup:1.0</nc:capability>
		<nc:capability>urn:ietf:params:netconf:capability:interleave:1.0</nc:capability>
	</nc:capabilities>
</nc:hello>
`

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

func (n ncclient) SendHello() string {
	return n.Write(NETCONF_HELLO)
}

// TODO: use the xml module to add/remove rpc related tags
func (n ncclient) WriteRPC(line string) string {
	line = fmt.Sprintf("<rpc>%s</rpc>", line)
	return n.Write(line)
}

// TODO: return io.Reader here rather than a string
func (n ncclient) Write(line string) string {
	line = line + NETCONF_DELIM
	input := bytes.NewBufferString(line)
	_, err := n.sessionStdin.Write(input.Bytes())
	if err != nil && err != io.EOF {
		panic(err)
	}

	xmlData := make([]string, 1)
	scanner := bufio.NewScanner(n.sessionStdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == NETCONF_DELIM {
			break
		}
		xmlData = append(xmlData, line)
	}
	return strings.Join(xmlData, "")
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
