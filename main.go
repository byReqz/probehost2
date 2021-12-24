package main
import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"net/http"
	"net"

	log "github.com/sirupsen/logrus"
)

var logstdout = log.New()
var logfile = log.New()

func init() {
	logstdout.SetFormatter(&log.TextFormatter{
		FullTimestamp: true})
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
	logfile.WithFields(log.Fields{
		"remote_ip": remoteip,
		"command": fmt.Sprint(command, args),
	}).Info("request initiated:")
	cmd, err := exec.Command(command, args...).Output()
	if err != nil {
		if ! strings.Contains(err.Error(), "1") {	// dont exit if error code is 1
			logstdout.WithFields(log.Fields{
				"remote_ip": remoteip,
				"command": fmt.Sprint(command, args),
				"error": err.Error(),
			}).Warn("request failed:")
			logfile.WithFields(log.Fields{
				"remote_ip": remoteip,
				"command": fmt.Sprint(command, args),
				"error": err.Error(),
			}).Warn("request failed:")
		}
	} else {
		logfile.WithFields(log.Fields{
			"remote_ip": remoteip,
			"command": fmt.Sprint(command, args),
		}).Info("request succeeded:")
	}
	return string(cmd)
}

func showhelp(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "placeholder")
}

func validatehosts(hosts []string) []string{
	var valid []string
	for _, host := range hosts {
		if net.ParseIP(host) != nil {
			valid = append(valid, host)
		} else if _, err := net.LookupIP(host); err == nil {
			valid = append(valid, host)
		}
	}
	return valid
}

func ping(w http.ResponseWriter, req *http.Request) {
	geturl := strings.Split(req.URL.String(), "/")
	targets := strings.Split(geturl[2], ",")
	hosts := validatehosts(targets)
	var res string
	for _, host := range hosts {
		res = fmt.Sprint(res, runner(req.RemoteAddr, "ping", "-c10", host), "\n")
	}
	if res == "" {
		fmt.Fprintln(w, http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, strings.TrimSpace(res), "\n")
	}
}

func mtr(w http.ResponseWriter, req *http.Request) {
	geturl := strings.Split(req.URL.String(), "/")
	targets := strings.Split(geturl[2], ",")
	hosts := validatehosts(targets)
	var res string
	for _, host := range hosts {
		res = fmt.Sprint(res, runner(req.RemoteAddr, "mtr", "-c10", "-w", host), "\n")
	}
	if res == "" {
		fmt.Fprintln(w, http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, strings.TrimSpace(res), "\n")
	}
}

func main() {
	http.HandleFunc("/ping/", ping)
	http.HandleFunc("/mtr/", mtr)
	http.HandleFunc("/", showhelp)
	fmt.Println("Serving on :8000")
	http.ListenAndServe(":8000", nil)
}