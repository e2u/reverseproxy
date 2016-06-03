package main

import (
  "encoding/hex"
  "flag"
  "fmt"
  "log"
  "net"
  "os"
  "time"
)

const BUF_SIZE = 0xffff

var (
  local_address       string
  remote_address      string
  remote_read_timeout int = 30 //远程连接超时时间,单位: 秒
  log_file            string
  access_log          *log.Logger
  access_id           int = 0
)

func init() {
  flag.Usage = show_usage
  flag.StringVar(&local_address, "l", "localhost:9000", `local listen local ip:port`)
  flag.StringVar(&remote_address, "r", "remote:9000", `remote ip and port  ip:port`)
  flag.IntVar(&remote_read_timeout, "t", 30, "remote connection timeout: default 30 sec")
  flag.StringVar(&log_file, "log", "", `dump to file /tmp/reverseproxy.log`)
  flag.Parse()
  initLogger()
}

func initLogger() {
  if len(log_file) == 0 {
    return
  }
  if out, err := os.OpenFile(log_file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModeAppend|0666); err == nil {
    access_log = log.New(out, "", 0)
  }
}

func main() {
  server, err := initListener(local_address)
  if err != nil || server == nil {
    panic(fmt.Sprintf("ERROR: couldn't start listening"))
  }
  fmt.Println("listen on", local_address, ",remote address -> ", remote_address)
  for {
    go handleConn(<-clientConns(server))
  }

}

//读取客户端发送来的消息,转发到远端服务器
func handleConn(local_conn net.Conn) {
  remote_conn, err := openConnect(remote_address)
  if err != nil {
    return
  }
  //关闭远程连接和本地连接
  defer remote_conn.Close()
  defer local_conn.Close()

  Pipe(local_conn, remote_conn)

}

func clientConns(listenner net.Listener) chan net.Conn {
  ch := make(chan net.Conn)
  go func() {
    if client, err := listenner.Accept(); err == nil && client != nil {
      access_id++
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
    access_log.Println("ERROR: Dial failed:", err.Error())
    return nil, err
  }
  conn.SetKeepAlive(true)
  return conn, nil
}

func show_usage() {
  fmt.Fprintf(os.Stderr,
    "Usage: %s \n",
    os.Args[0])
  flag.PrintDefaults()
}

func chanFromConn(conn net.Conn) chan []byte {
  c := make(chan []byte)
  go func() {
    b := make([]byte, BUF_SIZE)
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
      out_str := fmt.Sprintf("id: %09d,%v,LOCAL>>>>>\n%s%s\n%s\n", access_id, time.Now(), hex.Dump(b1), hex.EncodeToString(b1),string(b1))
      fmt.Print(out_str)
      if access_log != nil {
        access_log.Print(out_str)
      }
      remote.Write(b1)

    case b2 := <-remote_chan:
      if b2 == nil {
        return
      }
      out_str := fmt.Sprintf("id: %09d,%v,REMOTE<<<<<\n%s%s\n%s\n", access_id, time.Now(), hex.Dump(b2), hex.EncodeToString(b2),string(b2))
      fmt.Print(out_str)
      if access_log != nil {
        access_log.Print(out_str)
      }
      local.Write(b2)
    }
  }
}
