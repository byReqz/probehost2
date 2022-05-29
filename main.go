package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var logstdout = log.New()
var logfile = log.New()

var listenport int
var disablexforwardedfor bool
var allowprivate bool

func init() {
	logstdout.SetFormatter(&log.TextFormatter{
		FullTimestamp: true})
	logstdout.SetOutput(os.Stdout)
	logstdout.SetLevel(log.InfoLevel)
	var logfilepath string

	if _, exists := os.LookupEnv("PROBEHOST_LOGPATH"); exists == true {
		logfilepath, _ = os.LookupEnv("PROBEHOST_LOGPATH")
	} else {
		logfilepath = "probehost2.log"
	}
	if exists, _ := os.LookupEnv("PROBEHOST_ALLOW_PRIVATE"); exists == "true" {
		allowprivate = true
	} else {
		allowprivate = false
	}
	if envvalue, exists := os.LookupEnv("PROBEHOST_LISTEN_PORT"); exists == true {
		var err error
		listenport, err = strconv.Atoi(envvalue)
		if err != nil {
			logstdout.Fatal("Failed to read PROBEHOST_LISTEN_PORT: ", err.Error())
		}
	} else {
		listenport = 8000
	}
	if exists, _ := os.LookupEnv("PROBEHOST_DISABLE_X_FORWARDED_FOR"); exists == "true" {
		disablexforwardedfor = true
	} else {
		disablexforwardedfor = false
	}
	flag.StringVarP(&logfilepath, "logfilepath", "o", logfilepath, "sets the output file for the log")
	flag.IntVarP(&listenport, "port", "p", listenport, "sets the port to listen on")
	flag.BoolVarP(&disablexforwardedfor, "disable-x-forwarded-for", "x", disablexforwardedfor, "whether to show x-forwarded-for or the requesting IP")
	flag.BoolVarP(&allowprivate, "allow-private", "l", allowprivate, "whether to show lookups of private IP ranges")
	flag.Parse()

	logpath, err := os.OpenFile(logfilepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		logstdout.Fatal("Failed to initialize the logfile: ", err.Error())
	}
	logfile.SetLevel(log.InfoLevel)
	logfile.SetOutput(logpath)
	logfile.Info("probehost2 initialized")
}

func runner(remoteip string, command string, args ...string) string {
	logfile.WithFields(log.Fields{
		"remote_ip": remoteip,
		"command":   fmt.Sprint(command, args),
	}).Info("request initiated:")
	cmd, err := exec.Command(command, args...).Output()
	if err != nil {
		logstdout.WithFields(log.Fields{
			"remote_ip": remoteip,
			"command":   fmt.Sprint(command, args),
			"error":     err.Error(),
		}).Warn("request failed:")
		logfile.WithFields(log.Fields{
			"remote_ip": remoteip,
			"command":   fmt.Sprint(command, args),
			"error":     err.Error(),
		}).Warn("request failed:")
	} else {
		logfile.WithFields(log.Fields{
			"remote_ip": remoteip,
			"command":   fmt.Sprint(command, args),
		}).Info("request succeeded:")
	}
	return string(cmd)
}

func validatehosts(hosts []string) ([]string, []string) {
	var validhosts []string
	var validports []string
	for _, host := range hosts {
		split := strings.Split(host, "_")
		host = split[0]
		if hostparse := net.ParseIP(host); hostparse != nil {
			if (net.IP.IsPrivate(hostparse) || net.IP.IsLoopback(hostparse)) && allowprivate {
				validhosts = append(validhosts, host)
			} else if !(net.IP.IsPrivate(hostparse) || net.IP.IsLoopback(hostparse)) {
				validhosts = append(validhosts, host)
			}
		} else if _, err := net.LookupIP(host); err == nil {
			validhosts = append(validhosts, host)
		} else {
			continue
		}

		var port string
		if len(split) > 1 {
			port = split[1]
			_, err := strconv.Atoi(port) // validate if port is just an int
			if err == nil {
				validports = append(validports, port)
			} else {
				validports = append(validports, "0")
			}
		} else {
			validports = append(validports, "0")
		}
	}
	return validhosts, validports
}

func parseopts(options []string, cmdopts map[string]string) []string {
	var opts []string
	for _, opt := range options {
		opts = append(opts, cmdopts[opt])
	}
	return opts
}

func prerunner(req *http.Request, cmd string, cmdopts map[string]string, defaultopts []string) string {
	geturl := strings.Split(req.URL.String(), "/")
	targets := strings.Split(geturl[2], ",")
	hosts, ports := validatehosts(targets)
	var opts []string
	opts = append(opts, defaultopts...)
	if len(geturl) > 3 && len(geturl[3]) > 0 {
		options := strings.Split(geturl[3], ",")
		opts = append(opts, parseopts(options, cmdopts)...)
	}
	var res string
	var args []string
	var remoteaddr string
	if req.Header.Get("X-Forwarded-For") != "" && disablexforwardedfor != true {
		remoteaddr = req.Header.Get("X-Forwarded-For")
	} else {
		remoteaddr = req.RemoteAddr
	}
	for i, host := range hosts {
		runargs := append(args, opts...)
		if ports[i] != "0" && cmd == "nping" {
			runargs = append(runargs, "-p"+ports[i])
		}
		runargs = append(runargs, host)
		res = fmt.Sprint(res, runner(remoteaddr, cmd, runargs...), "\n")
	}
	return res
}

func ping(w http.ResponseWriter, req *http.Request) {
	cmd := "ping"
	cmdopts := map[string]string{
		"4": "-4", "6": "-6", "d": "-D", "n": "-n", "v": "-v", "c1": "-c1", "c5": "-c5", "c10": "-c10",
		"force4": "-4", "force6": "-6", "timestamps": "-D", "nodns": "-n", "verbose": "-v", "count1": "-c1", "count5": "-c5", "count10": "-c10",
	}
	var defaultopts []string
	defaultopts = append(defaultopts, "-c10")
	res := prerunner(req, cmd, cmdopts, defaultopts)
	if strings.TrimSpace(res) == "" {
		http.Error(w, "500: Internal Server Error", http.StatusInternalServerError)
	} else {
		_, _ = fmt.Fprint(w, strings.TrimSpace(res), "\n")
	}
}

func mtr(w http.ResponseWriter, req *http.Request) {
	cmd := "mtr"
	cmdopts := map[string]string{
		"4": "-4", "6": "-6", "u": "-u", "t": "-T", "e": "-e", "x": "-x", "n": "-n", "b": "-b", "z": "-z", "c1": "-c1", "c5": "-c5", "c10": "-c10",
		"force4": "-4", "force6": "-6", "udp": "-u", "tcp": "-T", "ext": "-e", "xml": "-x", "nodns": "-n", "cmb": "-b", "asn": "-z", "count1": "-c1", "count5": "-c5", "count10": "-c10",
	}
	var defaultopts []string
	defaultopts = append(defaultopts, "-r", "-w", "-c10")
	res := prerunner(req, cmd, cmdopts, defaultopts)
	if strings.TrimSpace(res) == "" {
		http.Error(w, "500: Internal Server Error", http.StatusInternalServerError)
	} else {
		_, _ = fmt.Fprint(w, strings.TrimSpace(res), "\n")
	}
}

func traceroute(w http.ResponseWriter, req *http.Request) {
	cmd := "traceroute"
	cmdopts := map[string]string{
		"4": "-4", "6": "-6", "f": "-F", "i": "-I", "t": "-T", "n": "-n", "u": "-U", "ul": "-UL", "d": "-D", "b": "--back",
		"force4": "-4", "force6": "-6", "dnf": "-F", "icmp": "-I", "tcp": "-T", "nodns": "-n", "udp": "-U", "udplite": "-UL", "dccp": "-D", "back": "--back",
	}
	var defaultopts []string
	//defaultopts = append(defaultopts) // no default options for traceroute
	res := prerunner(req, cmd, cmdopts, defaultopts)
	if strings.TrimSpace(res) == "" {
		http.Error(w, "500: Internal Server Error", http.StatusInternalServerError)
	} else {
		_, _ = fmt.Fprint(w, strings.TrimSpace(res), "\n")
	}
}

func nping(w http.ResponseWriter, req *http.Request) {
	cmd := "nping"
	cmdopts := map[string]string{
		"4": "-4", "6": "-6", "u": "--udp", "t": "--tcp-connect", "v": "-v", "c1": "-c1", "c3": "-c3", "c5": "-c5",
		"force4": "-4", "force6": "-6", "udp": "--udp", "tcp": "--tcp-connect", "verbose": "-v", "count1": "-c1", "count3": "-c3", "count5": "-c5",
	}
	var defaultopts []string
	defaultopts = append(defaultopts, "-c3")
	res := prerunner(req, cmd, cmdopts, defaultopts)
	if strings.TrimSpace(res) == "" {
		http.Error(w, "500: Internal Server Error", http.StatusInternalServerError)
	} else {
		_, _ = fmt.Fprint(w, strings.TrimSpace(res), "\n")
	}
}

func main() {
	http.HandleFunc("/ping/", ping)
	http.HandleFunc("/mtr/", mtr)
	http.HandleFunc("/tracert/", traceroute)
	http.HandleFunc("/traceroute/", traceroute)
	http.HandleFunc("/nping/", nping)
	logstdout.Info("Serving on :", listenport)
	logfile.Info("Serving on :", listenport)
	_ = http.ListenAndServe(fmt.Sprint(":", listenport), nil)
}
