package main
import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"net/http"

	log "github.com/sirupsen/logrus"
)

var logstdout = log.New()
var logfile = log.New()

func init() {
	logstdout.SetOutput(os.Stdout)
	logstdout.SetLevel(log.WarnLevel)

	logpath, err := os.OpenFile("probehost2.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		logstdout.Fatal("Failed to initialize the logfile: ", err.Error())
	}
	logfile.SetLevel(log.InfoLevel)
	logfile.SetOutput(logpath)
	logfile.Info("probehost2 initialized")
}

func runner(remoteip string, command string, args... string) string{
	cmd, err := exec.Command(command, args...).Output()
	if err != nil {
		if ! strings.Contains(err.Error(), "1") {	// dont exit if error code is 1
			logstdout.WithFields(log.Fields{
				"remote_ip": remoteip,
				"command": fmt.Sprint(command, args),
				"error": err.Error(),
			}).Warn("the following request failed:")
			logfile.WithFields(log.Fields{
				"remote_ip": remoteip,
				"command": fmt.Sprint(command, args),
				"error": err.Error(),
			}).Warn("the following request failed:")
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
	pingres := runner(req.RemoteAddr, "ping", "-c5", target)
	if pingres != "" {
		fmt.Fprintln(w, pingres)
	} else {
		fmt.Fprintln(w, http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/ping/", ping)
	http.HandleFunc("/", showhelp)
	fmt.Println("Serving on :8000")
	http.ListenAndServe(":8000", nil)
}