package ncclient

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
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

type Ncclient struct {
	username string
	password string
	hostname string
	key      string
	port     int

	sshClient     *ssh.ClientConn
	session       *ssh.Session
	sessionStdin  io.WriteCloser
	sessionStdout io.Reader
}

func (n Ncclient) Hostname() string {
	return n.hostname
}

func (n Ncclient) Close() {
	n.session.Close()
	n.sshClient.Close()
}

func (n Ncclient) SendHello() io.Reader {
	return n.Write(NETCONF_HELLO)
}

// TODO: use the xml module to add/remove rpc related tags
func (n Ncclient) WriteRPC(line string) io.Reader {
	line = fmt.Sprintf("<rpc>%s</rpc>", line)
	return n.Write(line)
}

func (n Ncclient) Write(line string) io.Reader {
	if _, err := io.WriteString(n.sessionStdin, line+NETCONF_DELIM); err != nil {
		panic(err)
	}

	xmlBuffer := bytes.NewBufferString("")
	scanner := bufio.NewScanner(n.sessionStdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == NETCONF_DELIM {
			break
		}
		xmlBuffer.WriteString(line + "\n")
	}
	return xmlBuffer
}

func MakeSshClient(username string, password string, hostname string, key string, port int) (*ssh.ClientConn, *ssh.Session, io.WriteCloser, io.Reader) {

	var config *ssh.ClientConfig

	if key != "" {
		block, _ := pem.Decode([]byte(key))
		if block == nil {
			panic("Impropery formatted private key received!")
		}

		rsakey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
		clientKey := &keychain{rsakey}

		config = &ssh.ClientConfig{
			User: username,
			Auth: []ssh.ClientAuth{
				ssh.ClientAuthKeyring(clientKey),
				ssh.ClientAuthPassword(clientPassword(password)),
			},
		}
	} else {
		config = &ssh.ClientConfig{
			User: username,
			Auth: []ssh.ClientAuth{
				ssh.ClientAuthPassword(clientPassword(password)),
			},
		}
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", hostname, strconv.Itoa(port)), config)
	if err != nil {
		panic("Failed to dial:" + hostname + err.Error())
	}

	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		panic(err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		panic(err)
	}
	return client, session, stdin, stdout
}

func (n *Ncclient) Connect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = errors.New(r.(string))
		}
	}()
	sshClient, sshSession, sessionStdin, sessionStdout := MakeSshClient(n.username, n.password, n.hostname, n.key, n.port)

	if err := sshSession.RequestSubsystem("netconf"); err != nil {
		// TODO: the command `xml-mode netconf need-trailer` can be executed
		// as a  backup if the netconf subsystem is not available, try that if we fail
		sshClient.Close()
		sshSession.Close()
		panic("Failed to make subsystem request: " + err.Error())
	}
	n.sshClient = sshClient
	n.session = sshSession
	n.sessionStdin = sessionStdin
	n.sessionStdout = sessionStdout
	return
}

func MakeClient(username string, password string, hostname string, key string, port int) Ncclient {
	nc := new(Ncclient)
	nc.username = username
	nc.password = password
	nc.hostname = hostname
	nc.key = key
	nc.port = port
	return *nc
}

type keychain struct {
	key *rsa.PrivateKey
}

func (k *keychain) Key(i int) (ssh.PublicKey, error) {
	if i != 0 {
		return nil, nil
	}
	return ssh.NewPublicKey(&k.key.PublicKey)
}

func (k *keychain) Sign(i int, rand io.Reader, data []byte) (sig []byte, err error) {
	hashFunc := crypto.SHA1
	h := hashFunc.New()
	h.Write(data)
	digest := h.Sum(nil)
	return rsa.SignPKCS1v15(rand, k.key, hashFunc, digest)
}
