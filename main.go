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

var logStdout = log.New()
var logFile = log.New()

var listenPort = 8080         // port to listen on
var disableXForwardedFor bool // whether to disable parsing the X-Forwarded-For header or not
var allowPrivate bool         // whether to allow private IP ranges or not

func init() {
	logStdout.SetFormatter(&log.TextFormatter{
		FullTimestamp: true})
	logStdout.SetOutput(os.Stdout)
	logStdout.SetLevel(log.InfoLevel)

	logFilePath := "probehost2.log"
	if val, exists := os.LookupEnv("PROBEHOST_LOGPATH"); exists {
		logFilePath = val
	}

	_, allowPrivate = os.LookupEnv("PROBEHOST_ALLOW_PRIVATE")
	_, disableXForwardedFor = os.LookupEnv("PROBEHOST_DISABLE_X_FORWARDED_FOR")

	if val, exists := os.LookupEnv("PROBEHOST_LISTEN_PORT"); exists {
		var err error
		listenPort, err = strconv.Atoi(val)
		if err != nil {
			logStdout.Fatal("Failed to read PROBEHOST_LISTEN_PORT: ", err.Error())
		}
	}

	flag.StringVarP(&logFilePath, "logFilePath", "o", logFilePath, "sets the output file for the log")
	flag.IntVarP(&listenPort, "port", "p", listenPort, "sets the port to listen on")
	flag.BoolVarP(&disableXForwardedFor, "disable-x-forwarded-for", "x", disableXForwardedFor, "whether to show x-forwarded-for or the requesting IP")
	flag.BoolVarP(&allowPrivate, "allow-private", "l", allowPrivate, "whether to show lookups of private IP ranges")
	flag.Parse()

	logpath, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		logStdout.Fatal("Failed to initialize the logFile: ", err.Error())
	}
	logFile.SetLevel(log.InfoLevel)
	logFile.SetOutput(logpath)
	logFile.Info("probehost2 initialized")
}

// runner runs the given command with the given args and returns stdout as string. Also logs all executed commands and their exit state.
func runner(remoteip string, command string, args ...string) string {
	logFile.WithFields(log.Fields{
		"remote_ip": remoteip,
		"command":   fmt.Sprint(command, args),
	}).Info("request initiated:")
	cmd, err := exec.Command(command, args...).Output()
	if err != nil {
		logStdout.WithFields(log.Fields{
			"remote_ip": remoteip,
			"command":   fmt.Sprint(command, args),
			"error":     err.Error(),
		}).Warn("request failed:")
		logFile.WithFields(log.Fields{
			"remote_ip": remoteip,
			"command":   fmt.Sprint(command, args),
			"error":     err.Error(),
		}).Warn("request failed:")
	} else {
		logFile.WithFields(log.Fields{
			"remote_ip": remoteip,
			"command":   fmt.Sprint(command, args),
		}).Info("request succeeded:")
	}
	return string(cmd)
}

// validatehosts checks the given host+port combinations for validity and returns valid hosts + valid ports separately.
func validatehosts(hosts []string) ([]string, []string) {
	var validHosts []string
	var validPorts []string
	for _, host := range hosts {
		split := strings.Split(host, "_")
		host = split[0]
		if hostparse := net.ParseIP(host); hostparse != nil {
			if (net.IP.IsPrivate(hostparse) || net.IP.IsLoopback(hostparse)) && allowPrivate {
				validHosts = append(validHosts, host)
			} else if !(net.IP.IsPrivate(hostparse) || net.IP.IsLoopback(hostparse)) {
				validHosts = append(validHosts, host)
			}
		} else if _, err := net.LookupIP(host); err == nil {
			validHosts = append(validHosts, host)
		} else {
			continue
		}

		var port string
		if len(split) > 1 {
			port = split[1]
			_, err := strconv.Atoi(port) // validate if port is just an int
			if err == nil {
				validPorts = append(validPorts, port)
			} else {
				validPorts = append(validPorts, "0")
			}
		} else {
			validPorts = append(validPorts, "0")
		}
	}
	return validHosts, validPorts
}

// parseopts matches the given user options to the valid optionmap.
func parseopts(options []string, cmdopts map[string]string) []string {
	var opts []string
	for _, opt := range options {
		opts = append(opts, cmdopts[opt])
	}
	return opts
}

// prerunner processes the incoming request to send it to runner.
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
	remoteaddr := req.RemoteAddr
	if req.Header.Get("X-Forwarded-For") != "" && !disableXForwardedFor {
		remoteaddr = req.Header.Get("X-Forwarded-For")
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

// ping is the response handler for the ping command. It defines the allowed options.
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

// mtr is the response handler for the mtr command. It defines the allowed options.
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

// traceroute is the response handler for the traceroute command. It defines the allowed options.
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

// nping is the response handler for the nping command. It defines the allowed options.
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
	logStdout.Info("Serving on :", listenPort)
	logFile.Info("Serving on :", listenPort)
	_ = http.ListenAndServe(fmt.Sprint(":", listenPort), nil)
}
