package gateway

import (
    "encoding/xml"
    "os"
    "fmt"
    "time"
    "strconv"
    "sync"
    "net/http"
    "log"
 //   "syscall"
)
//----------------------------following vars make a gateway-----------------//
var    ServerIp [4]byte
var ServerUrl string ="http://192.168.20.107:10008/"
var weblog *log.Logger
// IP should be written into config script
// User should be able to config it. New configuration should also be written into the config script
var Port =make([]PortType,6)

const (
    Ck=1
    Rfrsh=2
    Stop=3
)



func StopAll() {
    for _,p:=range Port {
        if p.Inuse&&p.Running {
            p.C<-Stop
        }
    }
}

func StartAll() {
    for _,p:=range Port {
        if p.Inuse {
            p.Init(p.PortID)
            p.ParsePortConf()
            go p.Run()
        }
    }
}
//---------------------------------------------------------------------------------------

type PortType struct {
    C chan byte
    PortID byte
    PortConf //niming
    Inuse bool  //if port is in use
    LastFileTime time.Time   // keep the last time.Time when db is filed.
    CurrentTimeString string
    CurrentDBName string//name of the db currently operating on
    ConfFileName string
    Running bool //if port is running rutine

    DbReadyList map[string]bool // a list of ready db files

    TTYLock sync.Mutex  //you have to get the lock to use tty
    TTYDev string
    Geter
}

type Geter interface {
    GetData() (float64,bool)
}

type PortConf struct {
    CheckPeriod int
    FilePeriod int
    Meters []MeterType
}

type MeterType struct {
    Addr uint64  // meter addr number
    PTypeID int //
    MTypeID int
    SHL  float64 //Super High Limit
    HL   float64//High Limit
    LL   float64//Low Limit
    SLL  float64//Super Low Limit
}



func (p * PortType)Init(portid byte) {
    p.C=make(chan byte)
    p.Inuse=true
    p.PortID=portid
    p.CheckPeriod=5 // 5 seconds
    p.FilePeriod=60 // 1 hour
    t:=time.Now()
    minutesoffzero:=t.Hour()*60+t.Minute()
    lastsavetimeinminutes:=minutesoffzero-minutesoffzero%p.FilePeriod
    houroff:=lastsavetimeinminutes/60
    minuteoff:=lastsavetimeinminutes%60
    duration:=(houroff-t.Hour())*60+minuteoff-t.Minute()
    p.LastFileTime=t.Add(time.Duration(duration)*time.Minute)
    p.CurrentTimeString=Time2Str(p.LastFileTime)
    p.DbReadyList=make(map[string]bool)
    p.CurrentDBName=strconv.FormatUint(uint64(portid),10)+"_"+p.CurrentTimeString+".csv"
    p.ConfFileName=strconv.FormatUint(uint64(portid),10)+".conf"
    p.TTYDev="/dev/ttyS"+strconv.FormatUint(uint64(portid),10)
}

func (p *PortType) RefreshConf() {
    p.C<-Rfrsh
}

func (p *PortType)Run(){
    // start ticker
    go p.StartTicker()
    p.Running=true
    for {//---------------------------------------------------big loop
        cmd:=<-p.C //look what cmd it is
         // CMD==Stop
        if cmd==Stop {
            p.Running=false
            return
        }
        // CMD=Rfrsh
        if cmd==Rfrsh {
            if p.ParsePortConf()!=true {
                //TODO this would not possibly happen, if it happens reports unrecoverable err.
                fmt.Println("Critical Error: ParseXml failed.")
                return
            }
        }
        var file *os.File
        var err error
        //open file for data
        file,err=os.OpenFile("/www/"+p.CurrentDBName,os.O_RDWR|os.O_CREATE|os.O_APPEND,0666)
        if err!=nil {
         // TODO  this would not possibly happen, if it happens reports unrecoverable err.
             fmt.Printf("Critical Error: Open %s.csv failed.",p.CurrentDBName)
             return
        }
        defer file.Close()
        //write to file data acquisition time
        _,err=file.WriteString(time.Now().Format("2006-01-02 15:04:05")+",")
        if err!=nil {
            //TODO this would not possibly happen, if it happens reports unrecoverable err.
        }
        for _,m:=range p.Meters{
            // construct a meter struct for 645 usage
            switch m.PTypeID {
                case 1:
                    p.Geter=&P645Geter{
                        Addr:m.Addr,
                        MTypeID:m.MTypeID,
                        Dev:p.TTYDev,
                    }
                case 2:
                    p.Geter=&PModbusGeter{
                        Addr:byte(m.Addr),
                        MTypeID:m.MTypeID,
                        Dev:p.TTYDev,
                    }
                default:
                    p.Geter=&P645Geter{
                        Addr:m.Addr,
                        MTypeID:m.MTypeID,
                        Dev:p.TTYDev,
                    }
            }
            p.TTYLock.Lock()
            res,ok:=p.GetData()
            p.TTYLock.Unlock()
            if !ok  {
                weblog.Println("gateway error 5")
		http.Head(ServerUrl+"alarm/alarm.aspx?Mac="+strconv.FormatUint(m.Addr,10)+"&errorid=5")
                res=0.0
            }
            fmt.Println(res)
            //do check 
            switch {
                case res>m.SHL:
                	weblog.Println("gateway error 7")
			http.Head(ServerUrl+"alarm/alarm.aspx?Mac="+strconv.FormatUint(m.Addr,10)+"&errorid=7")
                case res>m.HL:
                    //TODO signal outof limit
			weblog.Println("gateway error 8")
			http.Head(ServerUrl+"alarm/alarm.aspx?Mac="+strconv.FormatUint(m.Addr,10)+"&errorid=8")
                case res<m.SLL:
                    //TODO signal outof limit
			weblog.Println("gateway error 9")
			http.Head(ServerUrl+"alarm/alarm.aspx?Mac="+strconv.FormatUint(m.Addr,10)+"&errorid=9")
                case res<m.LL:
                    //TODO signal outof limit
			weblog.Println("gateway error 10")
			http.Head(ServerUrl+"alarm/alarm.aspx?Mac="+strconv.FormatUint(m.Addr,10)+"&errorid=10")
                default:
                    fmt.Println("checked, no problem.")
            }//end switch
            //write data into file
            file.WriteString(strconv.FormatUint(m.Addr,10)+":"+strconv.FormatFloat(res,'f',2,64)+",")
        }
        _,err=file.WriteString("\n")
        if err!=nil {
            fmt.Println(err)
            return
        }
        file.Close()
    }//-------------------------------------------------------------
}

// Two fields in the conf file are used.
// CheckPeriod: check and save data
// FilePeriod:  switch to an new database file
// do check anyhow and change file name string if it is time.
func (p *PortType)StartTicker() {
    //Ticks every CheckPeriod seconds.
    for t:=range time.Tick(time.Duration(p.CheckPeriod)*time.Second) {
        // Test if we have been saving data in this file for FilePeriod
        fmt.Println(t)
        if (t.Hour()-p.LastFileTime.Hour())*60+(t.Minute()-p.LastFileTime.Minute())==p.FilePeriod ||(t.Hour()==0 &&t.Minute()==0){
            //put this file in db ready list
            p.DbReadyList[p.CurrentTimeString]=true
            //change file name string
            p.CurrentTimeString=Time2Str(t)
            p.CurrentDBName=strconv.FormatUint(uint64(p.PortID),10)+"_"+p.CurrentTimeString+".csv"
            //update LastFileTime
            p.LastFileTime=t
            fmt.Println(p.DbReadyList)
            fmt.Println(p.CurrentTimeString)
        }
        //do check anyhow.
        if p.Running {
            p.C<-Ck
        }
    }
}

func (p * PortType)ParsePortConf() bool {
    xmlFile,err:=os.Open(p.ConfFileName)
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
    p.PortConf.Meters=nil
    err=xml.Unmarshal(fbyte,&p.PortConf)
    if err!=nil {
        fmt.Printf("error ummarshalling xml\n")
        return false
    }
    fmt.Println(p.PortConf)
    return true
}


func Time2Str(t time.Time) string {
    return fmt.Sprintf("%d%02d%02d%02d%02d",
                       t.Year(),
                       t.Month(),
                       t.Day(),
                       t.Hour(),
                       t.Minute())
}



