package main

import (
  "encoding/hex"
  "flag"
  "fmt"
  "net"
  "os"
)

var (
  local_address       string
  remote_address      string
  remote_read_timeout int = 30 //远程连接超时时间,单位: 秒
)

func init() {
  flag.Usage = show_usage
  flag.StringVar(&local_address, "l", "localhost:9000", `local listen local ip:port`)
  flag.StringVar(&remote_address, "r", "remote:9000", `remote ip and port  ip:port`)
  flag.IntVar(&remote_read_timeout, "t", 30, "remote connection timeout: default 30 sec")
  flag.Parse()
}

func main() {
  server, err := initListener(local_address)
  if err != nil || server == nil {
    panic(fmt.Sprintf("ERROR: couldn't start listening"))
  }
  conns := clientConns(server)
  for {
    go handleConn(<-conns)
  }
}

//读取客户端发送来的消息,转发到远端服务器
func handleConn(local_conn net.Conn) {

  remote_conn, err := openConnect(remote_address)
  //remote_conn.SetReadDeadline(time.Now().Add(time.Duration(remote_read_timeout) * time.Second))
  remote_conn.SetKeepAlive(true)
  remote_conn.SetNoDelay(true)
  if err != nil {
    return
  }
  //关闭远程连接和本地连接
  defer remote_conn.Close()
  defer local_conn.Close()

  Pipe(local_conn, remote_conn)
}

/*

*/
func clientConns(listenner net.Listener) chan net.Conn {
  ch := make(chan net.Conn)
  i := 0
  go func() {
    for {
      client, err := listenner.Accept()
      if client == nil {
        fmt.Printf("ERROR: couldn't accept: %v", err)
        continue
      }
      i++
      fmt.Printf("%d: %v <-> %v \n", i, client.LocalAddr(), client.RemoteAddr())
      ch <- client
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
    return nil, err
  }
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
    b := make([]byte, 1024)
    for {
      n, err := conn.Read(b)
      if n > 0 {
        res := make([]byte, n)
        copy(res, b[:n])
        c <- res
      }
      if err != nil {
        c <- nil
        break
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
      } else {
        fmt.Printf("LOCAL>>>>>\n%s", hex.Dump(b1))
        remote.Write(b1)
      }
    case b2 := <-remote_chan:
      if b2 == nil {
        return
      } else {
        fmt.Printf("REMOTE<<<<<\n%s", hex.Dump(b2))
        local.Write(b2)
      }
    }
  }
}
