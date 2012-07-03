package gateway
import (
     "fmt"
     "syscall"
     "os"
     "math"
     "time"
     "encoding/xml"
     "strconv"
)

const ConfDir="645/"

const (
    start byte=0
    a0 byte=1
    a1 byte=2
    a2 byte=3
    a3 byte=4
    a4 byte=5
    a5 byte=6
    restart byte=7
    ctrl byte=8
    length byte=9
    data0 byte=10
    data1 byte=11
    cs byte=12
    stop byte=13
)

type P645MConf struct {
    StartByte int //
    DataLength byte  //int
    IntegerLength byte //int
    DataID uint16  //hexidecimal
    Wait int   //in milliseconds
    TTYParamType
}

// type TTYParamType struct {
//     BaudRate uint16 //`xml:"SerialParams>BaudRate"`
//     NumStopBits byte
//     NumDataBits byte
//     Parity byte  //2 for Even, 1 for Odd, 0 for none
// }

type P645Geter struct {
    Addr uint64
    MTypeID int
    Dev string
    P645MConf  //niming
    txbuf []byte
    rxbuf []byte
}

func (g *P645Geter) fillCMD() (ok bool){
    g.txbuf=make([]byte,14)
    g.rxbuf=make([]byte,30)
    g.txbuf[start]=0x68
    tmp:=g.Addr
    for i:=0;i<5;i++ {
        g.txbuf[i+1]=byte(tmp%10)
        g.txbuf[i+1]+=byte(tmp%100/10*16)
        tmp=tmp/100
    }
    g.txbuf[restart]=0x68
    g.txbuf[ctrl]=0x01
    g.txbuf[length]=0x02
    g.txbuf[data0]=byte(g.DataID%10+0x33+g.DataID/10%10*16)
    g.txbuf[data1]=byte((g.DataID/100%10)+0x33+g.DataID/1000*16)
    g.txbuf[cs]=0
    g.txbuf[cs],ok=g.csCheck(g.txbuf,12)
    if ok==false {
        return
    }
    g.txbuf[13]=0x16
    return true
}

func (g *P645Geter) csCheck(buf []byte,n byte) (ret byte,ok bool) {
    if len(buf)<int(n) {return 0,false}
    ret=0
    for i:=byte(0);i<n;i++ {
        ret=byte(ret+buf[i])
    }
    ok=true
    return
}

func (g * P645Geter) ParseMeterConf() bool{
    xmlFile,err:=os.Open(ConfDir+strconv.FormatUint(uint64(g.MTypeID),10)+".conf")
    if err!=nil {
        return false
    }
    defer xmlFile.Close()
    FStat,err := xmlFile.Stat()
    if err!=nil {
        return false
    }
    fbyte:= make([]byte,FStat.Size())
    _,err = xmlFile.Read(fbyte)
    if err!=nil {
        return false
    }
    err=xml.Unmarshal(fbyte,&(g.P645MConf))
    if err!=nil {
        fmt.Printf("error ummarshalling xml\n")
        return false
    }
    return true
}

func (g *P645Geter) GetData() (res float64,ok bool) {
    //do serial init
    fmt.Println(g)
    g.ParseMeterConf()
    g.fillCMD()
//    fmt.Println(g)

    fd,err:=syscall.Open(g.Dev,os.O_RDWR,0666)
    if err!=nil {
        ok=false
        return
    }
    defer syscall.Close(fd)
    _,err=syscall.Write(fd,g.txbuf)
    if err!=nil {
        ok=false
        return
    }
    time.Sleep(time.Duration(g.Wait)*time.Millisecond)
    n,err:=syscall.Read(fd,g.rxbuf)
    if err!=nil||n<18 {
        fmt.Println(err)
        return 0,false
    }
    g.rxbuf=g.rxbuf[g.StartByte:] //where the real data starts

    cslen:=(g.rxbuf[length]>>4)&0xf*10+(g.rxbuf[length]&0xf)+length+1
    if (g.rxbuf[length]-2)<g.DataLength/2 {
        ok=false
        return
    }
    if ck,ok:=g.csCheck(g.rxbuf,cslen);g.rxbuf[cslen]!=ck||ok==false {
        return res,false
    }
    i:=cslen-1
    j:=int(g.IntegerLength)
    k:=g.DataLength
    for ;i>(length+2);{ //2 stands for DataID[2]
        if k==0 { break}
        j--
        res += float64((g.rxbuf[i]>>4)&0xf-0x3)*math.Pow10(j)
        j--
        k--
        if k==0 { break}
        res += float64(g.rxbuf[i]&0xf-0x3)*math.Pow10(j)
        i--
        k--
    }
    syscall.Close(fd)
    return res,true    
}

