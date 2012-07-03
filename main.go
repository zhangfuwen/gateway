package main
import (
//     "encoding/xml"
//     "os"
 //   "fmt"
    "gateway"
    "time"
    "webserver"
)


func main() {
    gateway.Port[4-1].Inuse=true
    //gateway.StartAll()
    p:=&gateway.Port[4-1]
    p.Init(4)
    p.ParsePortConf()
    go p.Run()

    
    go webserver.Serve()
    go webserver.FileServe()

    var ch=make(chan time.Time,5) // this and the following line just make a
    <-ch                          //simple trick to block main
}
