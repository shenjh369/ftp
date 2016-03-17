// author: smallfish <smallfish.xy@gmail.com>

package ftp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
    "os"
    "bufio"
    "io"
    "log"
    "github.com/toolkits/file"
)

func init() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

type FTP struct {
	host    string
	port    int
	user    string
	passwd  string
	pasv    int
	cmd     string
	Code    int
	Message string
	Debug   bool
	stream  []byte
	conn    net.Conn
	Error   error
}

func (ftp *FTP) debugInfo(s string) {
	if ftp.Debug {
		log.Println(s)
	}
}

func (ftp *FTP) Connect(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	ftp.conn, ftp.Error = net.Dial("tcp", addr)
	ftp.Response()
	ftp.host = host
	ftp.port = port
}

func (ftp *FTP) Login(user, passwd string) {
	ftp.Request("USER " + user)
	ftp.Request("PASS " + passwd)
	ftp.user = user
	ftp.passwd = passwd
}

func (ftp *FTP) Response() (code int, message string) {
	ret := make([]byte, 1024)
	n, _ := ftp.conn.Read(ret)
	msg := string(ret[:n])
	code, _ = strconv.Atoi(msg[:3])
	message = msg[4 : len(msg)-2]
	ftp.debugInfo("<*cmd*> " + ftp.cmd)
	ftp.debugInfo(fmt.Sprintf("<*code*> %d", code))
	ftp.debugInfo("<*message*> " + message)
	return
}

func (ftp *FTP) Request(cmd string) {
	ftp.conn.Write([]byte(cmd + "\r\n"))
	ftp.cmd = cmd
	ftp.Code, ftp.Message = ftp.Response()
	if cmd == "PASV" {
		start, end := strings.Index(ftp.Message, "("), strings.Index(ftp.Message, ")")
		s := strings.Split(ftp.Message[start:end], ",")
		l1, _ := strconv.Atoi(s[len(s)-2])
		l2, _ := strconv.Atoi(s[len(s)-1])
		ftp.pasv = l1*256 + l2
	}
	if (cmd != "PASV") && (ftp.pasv > 0) {
		ftp.Message = newRequest(ftp.host, ftp.pasv, ftp.stream)
		ftp.debugInfo("<*response*> " + ftp.Message)
		ftp.pasv = 0
		ftp.stream = nil
		ftp.Code, _ = ftp.Response()
	}
}

func (ftp *FTP) Pasv() {
	ftp.Request("PASV")
}

func (ftp *FTP) Pwd() {
	ftp.Request("PWD")
}

func (ftp *FTP) Cwd(path string) {
	ftp.Request("CWD " + path)
}

func (ftp *FTP) Mkd(path string) {
	ftp.Request("MKD " + path)
}

func (ftp *FTP) Size(path string) (size int) {
	ftp.Request("SIZE " + path)
	size, _ = strconv.Atoi(ftp.Message)
	return
}

func (ftp *FTP) List() {
	ftp.Pasv()
	ftp.Request("LIST")
}

func (ftp *FTP) Stor(file string, data []byte) {
	ftp.Pasv()
	if data != nil {
		ftp.stream = data
	}
	ftp.Request("STOR " + file)
}

func (ftp *FTP) Retr(path, dest string) {
    ftp.Request("TYPE I")
    ftp.Pasv()

    cmd := "RETR " + path
    ftp.conn.Write([]byte(cmd + "\r\n"))
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ftp.host, ftp.pasv))
    if err != nil {
        log.Println("dial ftp fail", err.Error())
        return
    }
    defer conn.Close()

    os.MkdirAll(strings.TrimRight(dest, file.Basename(dest)), os.ModePerm)

    f, ferr := os.OpenFile(dest, os.O_CREATE | os.O_TRUNC, os.ModePerm)
    if ferr != nil {
        log.Println(ferr.Error())
        return
    }
    defer f.Close()

    buf := bufio.NewWriter(f)
    defer buf.Flush()

    for {
        ret := make([]byte, 4096)
        n, err := conn.Read(ret)
        if err == io.EOF {
            break
        }
        buf.Write(ret[:n])
    }
    ftp.Response()

    defer func() {
        ftp.pasv = 0
        ftp.stream = nil
    }()
}

func (ftp *FTP) Quit() {
	ftp.Request("QUIT")
	ftp.conn.Close()
}

// new connect to FTP pasv port, return data
func newRequest(host string, port int, b []byte) string {
	conn, _ := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	defer conn.Close()
	if b != nil {
		conn.Write(b)
		return "OK"
	}
	ret := make([]byte, 4096)
	n, _ := conn.Read(ret)
	return string(ret[:n])
}
