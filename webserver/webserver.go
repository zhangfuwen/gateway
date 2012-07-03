package webserver
import (
            "fmt"
            "net/http"
            "io"
            "syscall"
            "strconv"
            "os"
            "gateway"
	    "log"
       )

var lastHour uint64
var weblog * log.Logger


func reqHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println(r)
    fmt.Println(r.URL.Path)
//-------------------------------timesync done----------------------
//this may block for a minute
    if r.URL.Path=="/timesync.go"{ //only do this when gateway is not working.
        var tv syscall.Timeval
        num,err := fmt.Sscanf(r.FormValue("Sec"),"%d",&(tv.Sec))
        if err != nil ||num!=1||tv.Sec==0  {
            w.WriteHeader(400)  //bad request
            return
        }
        gateway.StopAll()// all ports stop working while doing time sync.
        tv.Usec=0
        syscall.Settimeofday(&tv)
        gateway.StartAll() //restart all
        w.WriteHeader(200) //OK ,time sync done!
        return
    }
//-------------------------------------------------------------------------
if r.URL.Path=="/fetchdata.go" {
    var t string
    var portId uint64
    _,err:=fmt.Sscanf(r.FormValue("portid"),"%d",&portId)
    if err!=nil {
        w.WriteHeader(400) //bad request 
        return
    }
    t=r.FormValue("time")
    if err!=nil {
        w.WriteHeader(400) //bad request 
        return
    }
    // check if file exists
    p:=&gateway.Port[portId-1]
    if _,ok:=p.DbReadyList[t];ok {
        filename:=strconv.FormatUint(portId,10)+"_"+t+".csv"
        w.Header().Add("Location",filename)
        w.WriteHeader(200)
        delete(p.DbReadyList,t) //delete this key after return
//TODO: When released as a product, the following line should be uncommented 
//        err=os.Remove("/www/"+filename)
    }else { //if not
        w.WriteHeader(400)
    }
        return
}
//--------------------fetchinstant.go?portId=1&slaveAddr=byte(0xF9)-------------
    if r.URL.Path=="/fetchinstant.go" {
        var portId int
        var slaveAddr uint64
        num,err:=fmt.Sscanf(r.FormValue("portid"),"%d",&portId)
        if err!=nil ||num!=1 ||portId>=8 ||portId<0{
             w.WriteHeader(400) //bad request
            return
        }
        num,err=fmt.Sscanf(r.FormValue("slaveaddr"),"%d",&slaveAddr)
        if err!=nil ||num!=1 ||slaveAddr<0 {
            w.WriteHeader(400) //bad request
            return
        }
        fmt.Println(slaveAddr)
        p:=&gateway.Port[portId-1]
        for _,m:=range p.Meters{
            if m.Addr==slaveAddr {
                fmt.Println(m)
                switch m.PTypeID {
                    case 1:
                        p.Geter=&gateway.P645Geter{
                            Addr:m.Addr,
                            MTypeID:m.MTypeID,
                            Dev:p.TTYDev,
                        }
                    default:
                        p.Geter=&gateway.P645Geter{
                            Addr:m.Addr,
                            MTypeID:m.MTypeID,
                            Dev:p.TTYDev,
                        }
                }
                p.TTYLock.Lock()
                res,ok:=p.GetData()
                fmt.Println(res)
                p.TTYLock.Unlock()
                if ok==true {
                    w.Header().Add("Data",strconv.FormatFloat(res,'f',2,64))
                    w.WriteHeader(200)
                    return
                }
            }

        }
        w.WriteHeader(400)
        return
    }
//--------------------configure.go-------------------------
    if r.URL.Path=="/configure.go" {
        var portId int
        num,err:=fmt.Sscanf(r.FormValue("portid"),"%d",&portId)
        fmt.Println(portId)
        if err!=nil ||num!=1 ||portId>=8 ||portId<0{
            w.WriteHeader(400) //bad request
	    weblog.Println("configure.go error 1")
            return
        }
        fn,_,err:=r.FormFile("File")
        defer fn.Close()
        if err!=nil {
            w.WriteHeader(400)
	    weblog.Println("configure.go error 2")
	    return
        }
        f,err:=os.Create(strconv.FormatInt(int64(portId),10)+".conf")
        defer f.Close()
        if err!=nil {
            w.WriteHeader(400)
	    weblog.Println("configure.go error 3")
	    return
        }
        io.Copy(f,fn)
        if err!=nil {
            w.WriteHeader(400)
	    weblog.Println("configure.go error 4")
	    return
        }
	weblog.Println("configure.go confchanged")
        if gateway.Port[portId-1].Inuse {
            gateway.Port[portId-1].RefreshConf()
	    weblog.Println("configure.go confrefreshed")
        }
        w.WriteHeader(200)
        //TODO signal gateway to refresh settings
    }
//----------------------warn.go------------------------------
// gateway do it
}//end reqHandler

func Serve() {
    lastHour=2012050612
	weblog=log.New(os.Stdout, "", log.Ldate|log.Ltime)
    http.HandleFunc("/", reqHandler)
    http.ListenAndServe(":80", nil)
}

func FileServe() {
    h:=http.FileServer(http.Dir("/www"))
    http.ListenAndServe(":8080",h)
}
