package main

import (
  "bufio"
  "encoding/hex"
  "fmt"
  "net"
  "os"
  "flag"
)


var local_address string
var remote_address string

func init(){
  flag.Usage = show_usage
  flag.StringVar(&local_address, "l", "localhost:9000", `local listen local ip:port`)
  flag.StringVar(&remote_address, "r", "remote:9000", `remote ip and port  ip:port`)
  flag.Parse()  
}


func main() {
  server := initListener(local_address)
  if server == nil {
    panic(fmt.Sprintf("couldn't start listening"))
  }
  conns := clientConns(server)
  for {
    go handleConn(<-conns)
  }
}

//读取客户端发送来的消息,转发到远端服务器
func handleConn(c net.Conn) {
  
  cb := readBufferd(bufio.NewReader(c))
  fmt.Printf("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<\n%v\n", hex.Dump(cb))

  rc := openConnect(remote_address)
  defer closeConnect(rc)

  rc.Write(cb)
  b := bufio.NewReader(rc)
  rrb := readBufferd(b)
  c.Write(rrb)
  fmt.Printf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\n%v\n", hex.Dump(rrb))
  
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
        fmt.Printf("couldn't accept: %v", err)
        continue
      }
      i++
      fmt.Printf("%d: %v <-> %v \n", i, client.LocalAddr(), client.RemoteAddr())
      ch <- client
    }
  }()
  return ch
}

func initListener(addr string) *net.TCPListener {
  s, err := net.ResolveTCPAddr("tcp", addr)
  if err != nil {
    panic(fmt.Sprintf("ResolveTCPAddr failed:%v", err))
  }
  l, err := net.ListenTCP("tcp", s)
  if err != nil {
    panic(fmt.Sprintf("can't listen on %s,%v", addr, err))
  }
  return l
}

func openConnect(addr string) *net.TCPConn {
  s, err := net.ResolveTCPAddr("tcp", addr)
  conn, err := net.DialTCP("tcp", nil, s)
  if err != nil {
    println("Dial failed:", err.Error())
  }
  return conn
}

func closeConnect(conn *net.TCPConn) {
  if &conn != nil {
    conn.Close()
  }
}

func readBufferd(r *bufio.Reader) []byte {
  r.Peek(1)
  len := r.Buffered()
  buf := make([]byte, len)
  r.Read(buf)
  return buf
}


func show_usage() {
  fmt.Fprintf(os.Stderr,
    "Usage: %s \n",
    os.Args[0])
  flag.PrintDefaults()
}


