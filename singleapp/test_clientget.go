package main
import (
	"net/http"
	"fmt"
)

func main () {
	res,_:=http.Head("http://192.168.20.107:10008/alarm/alarm.aspx?Mac=1204100008")
	fmt.Println(res)
	return
}
