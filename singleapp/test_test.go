package main
import (
    "os"
    "fmt"
)

func main() {
    fb,_:=os.OpenFile("a.txt",os.O_RDWR|os.O_CREATE,0644)
    buf:=[]byte("http://www.usr.cc")
    fb.Write(buf)
    rx_buf:=make([]byte,4)
    fb.ReadAt(rx_buf,0)
    fmt.Println(string(rx_buf))
    fb.Close()
}