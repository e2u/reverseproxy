package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type LoggerType uint16

const (
	FormatAll   LoggerType = 1 << iota // 0x01
	FormatFHex                         //0x02
	FormatHex                          //0x04
	FormatPlain                        //0x08
	BuffSize    = 0xffff
)

/*
logFormat 日志格式:
	all: 显示所有格式
	fhex: 格式化的16进制格式
	hex: 无格式 16进制格式
	plain: 文本
请求参数格式 -lf [all|fhex|hex|plain] 或 -lf fhex,plain 或 -lf plain
默认值: plain
*/
var (
	localAddress  string
	remoteAddress string
	remoteTimeout int = 30 //远程连接超时时间,单位: 秒
	logFile       string
	accessLog     *log.Logger
	accessID      int = 0
	logFormatFlag string
	logFormat     LoggerType
	fhexOn        bool
	hexOn         bool
	plainOn       bool
	allOn         bool
)

func init() {
	flag.Usage = showUsage
	flag.StringVar(&localAddress, "l", "localhost:9000", `local listen local ip:port`)
	flag.StringVar(&remoteAddress, "r", "remote:9000", `remote ip and port  ip:port`)
	flag.IntVar(&remoteTimeout, "t", 30, "remote connection timeout: default 30 sec")
	flag.StringVar(&logFile, "log", "", `dump to file /tmp/reverseproxy.log`)
	flag.StringVar(&logFormatFlag, "lf", "plain", "logger output format. [all|fhex,hex,plain] e.g. -lf=fhex,plain or -lf=fhex,hex or -lf=all")
	flag.Parse()
	initLogger()

	for _, f := range strings.Split(logFormatFlag, ",") {
		switch f {
		case "fhex":
			logFormat |= FormatFHex
		case "hex":
			logFormat |= FormatHex
		case "plain":
			logFormat |= FormatPlain
		case "all":
			logFormat |= FormatAll
		}
	}

	if logFormat&FormatHex == FormatHex {
		log.Println("Hex output is On")
		hexOn = true
	}

	if logFormat&FormatFHex == FormatFHex {
		log.Println("FHex output is On")
		fhexOn = true
	}

	if logFormat&FormatPlain == FormatPlain {
		log.Println("Plain output is On")
		plainOn = true
	}

	if logFormat&FormatAll == FormatAll {
		log.Println("All output is On")
		allOn = true
	}

}

func initLogger() {
	if len(logFile) == 0 {
		return
	}
	if out, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModeAppend|0644); err == nil {
		accessLog = log.New(out, "", 0)
	}
}

func main() {
	server, err := initListener(localAddress)
	if err != nil || server == nil {
		panic(fmt.Sprintf("ERROR: couldn't start listening"))
	}
	fmt.Println("listen on", localAddress, ",remote address -> ", remoteAddress)
	for {
		go handleConn(<-clientConns(server))
	}

}

//读取客户端发送来的消息,转发到远端服务器
func handleConn(localConn net.Conn) {
	remoteConn, err := openConnect(remoteAddress)
	if err != nil {
		return
	}
	//关闭远程连接和本地连接
	defer remoteConn.Close()
	defer localConn.Close()

	Pipe(localConn, remoteConn)

}

func clientConns(listenner net.Listener) chan net.Conn {
	ch := make(chan net.Conn)
	go func() {
		if client, err := listenner.Accept(); err == nil && client != nil {
			accessID++
			ch <- client
		} else {
			fmt.Printf("ERROR: couldn't accept: %v", err)
		}
	}()
	return ch
}

func initListener(addr string) (*net.TCPListener, error) {
	s, err := net.ResolveTCPAddr("tcp", addr)

	if err != nil {
		panic(fmt.Sprintf("ResolveTCPAddr failed:%v", err))
		return nil, err
	}

	l, err := net.ListenTCP("tcp", s)

	if err != nil {
		panic(fmt.Sprintf("can't listen on %s,%v", addr, err))
		return nil, err
	}

	return l, nil
}

func openConnect(addr string) (*net.TCPConn, error) {
	s, err := net.ResolveTCPAddr("tcp", addr)
	conn, err := net.DialTCP("tcp", nil, s)
	if err != nil {
		println("ERROR: Dial failed:", err.Error())
		accessLog.Println("ERROR: Dial failed:", err.Error())
		return nil, err
	}
	conn.SetKeepAlive(true)
	return conn, nil
}

func showUsage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s \n",
		os.Args[0])
	flag.PrintDefaults()
}

func chanFromConn(conn net.Conn) chan []byte {
	c := make(chan []byte)
	go func() {
		b := make([]byte, BuffSize)
		for {
			if n, err := conn.Read(b); err != nil {
				c <- nil
				break
			} else {
				c <- b[:n]
			}
		}
	}()
	return c
}

func Pipe(local net.Conn, remote net.Conn) {
	local_chan := chanFromConn(local)
	remote_chan := chanFromConn(remote)
	for {
		select {
		case b1 := <-local_chan:
			if b1 == nil {
				return
			}
			var outLine []string
			outLine = append(outLine, fmt.Sprintf("id: %09d,%v,LOCAL>>>>>>>>>>", accessID, time.Now()))
			if fhexOn || allOn {
				outLine = append(outLine, hex.Dump(b1))
			}
			if hexOn || allOn {
				outLine = append(outLine, hex.EncodeToString(b1))
			}
			if plainOn || allOn {
				outLine = append(outLine, string(b1))
			}

			outLine = append(outLine, "\n")

			fmt.Print(strings.Join(outLine, "\n"))

			if accessLog != nil {
				accessLog.Print(strings.Join(outLine, "\n"))
			}
			remote.Write(b1)

		case b2 := <-remote_chan:
			if b2 == nil {
				return
			}
			var outLine []string
			outLine = append(outLine, fmt.Sprintf("id: %09d,%v,REMOTE<<<<<<<<<<", accessID, time.Now()))
			if fhexOn || allOn {
				outLine = append(outLine, hex.Dump(b2))
			}
			if hexOn || allOn {
				outLine = append(outLine, hex.EncodeToString(b2))
			}
			if plainOn || allOn {
				outLine = append(outLine, string(b2))
			}

			outLine = append(outLine, "\n")

			fmt.Print(strings.Join(outLine, "\n"))

			if accessLog != nil {
				accessLog.Print(strings.Join(outLine, "\n"))
			}

			local.Write(b2)
		}
	}
}
