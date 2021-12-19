package main
import (
	"fmt"
	"os/exec"
	"log"
	"strings"
	"net/http"
)

func runner(command string, args... string) string{
	cmd, err := exec.Command(command, args...).Output()
	if err != nil {
		if ! strings.Contains(err.Error(), "1") {	// dont exit if error code is 1
			log.Fatal(command, args, "caused an error: ", err)
		}
	}
	return string(cmd)
}

func showhelp(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "placeholder")
}

func ping(w http.ResponseWriter, req *http.Request) {
	geturl := strings.Split(req.URL.String(), "/")
	target := geturl[2]
	pingres := runner("ping", "-c5", target)
	fmt.Fprintln(w, pingres)
}

func main() {
	http.HandleFunc("/ping/", ping)
	http.HandleFunc("/", showhelp)
	fmt.Println("Serving on :8000")
	http.ListenAndServe(":8000", nil)
}