package main
import (
  "fmt"
  "os"
  "os/exec"
  "strings"
  "net/http"
  "net"
  "flag"

  log "github.com/sirupsen/logrus"
)

var logstdout = log.New()
var logfile = log.New()

var listenport int

func init() {
  var logfilepath string
  flag.StringVar(&logfilepath, "logfilepath", "probehost2.log", "sets the output file for the log")
  flag.IntVar(&listenport, "port", 8000, "sets the port to listen on")
  flag.Parse()

  logstdout.SetFormatter(&log.TextFormatter{
    FullTimestamp: true})
  logstdout.SetOutput(os.Stdout)
  logstdout.SetLevel(log.InfoLevel)

  logpath, err := os.OpenFile(logfilepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
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
  } else {
    logfile.WithFields(log.Fields{
      "remote_ip": remoteip,
      "command": fmt.Sprint(command, args),
    }).Info("request succeeded:")
  }
  return string(cmd)
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

func parseopts(options []string, cmdopts map[string]string) []string{
  var opts []string
  for _, opt := range options {
    opts = append(opts, cmdopts[opt])
  }
  return opts
}

func prerunner(req *http.Request, cmd string, cmdopts map[string]string, defaultopts []string) string{
  geturl := strings.Split(req.URL.String(), "/")
  targets := strings.Split(geturl[2], ",")
  hosts := validatehosts(targets)
  var opts []string
  opts = append(opts, defaultopts...)
  if len(geturl) > 3 && len(geturl[3]) > 0 { 
    options := strings.Split(geturl[3], ",")
    opts = append(opts, parseopts(options, cmdopts)...)
  }
  var res string
  var args []string
  for _, host := range hosts {
    args = append(args, opts...)
    args = append(args, host)
    res = fmt.Sprint(res, runner(req.RemoteAddr, cmd, args...), "\n")
  }
  return res
}

func ping(w http.ResponseWriter, req *http.Request) {
  cmd := "ping"
  cmdopts := map[string]string{"4": "-4", "6": "-6", "d": "-D", "n": "-n", "v": "-v", "c1": "-c1", "c5": "-c5", "c10": "-c10"}
  var defaultopts []string
  defaultopts = append(defaultopts, "-c10")
  res := prerunner(req, cmd, cmdopts, defaultopts)
  if strings.TrimSpace(res) == "" {
    fmt.Fprintln(w, http.StatusInternalServerError)
  } else {
    fmt.Fprint(w, strings.TrimSpace(res), "\n")
  }
}

func mtr(w http.ResponseWriter, req *http.Request) {
  cmd := "mtr"
  cmdopts := map[string]string{"4": "-4", "6": "-6", "u": "-u", "t": "-T", "e": "-e", "x": "-x", "n": "-n", "b": "-b", "z": "-z", "c1": "-c1", "c5": "-c5", "c10": "-c10"}
  var defaultopts []string
  defaultopts = append(defaultopts, "-r", "-w", "-c10")
  res := prerunner(req, cmd, cmdopts, defaultopts)
  if strings.TrimSpace(res) == "" {
    fmt.Fprintln(w, http.StatusInternalServerError)
  } else {
    fmt.Fprint(w, strings.TrimSpace(res), "\n")
  }
}

func traceroute(w http.ResponseWriter, req *http.Request) {
  cmd := "traceroute"
  cmdopts := map[string]string{"4": "-4", "6": "-6", "dnf": "-F", "i": "-I", "t": "-T", "n": "-n", "u": "-U", "ul": "-UL", "d": "-D", "b": "--back"}
  var defaultopts []string
  //defaultopts = append(defaultopts) // no default options for traceroute
  res := prerunner(req, cmd, cmdopts, defaultopts)
  if strings.TrimSpace(res) == "" {
    fmt.Fprintln(w, http.StatusInternalServerError)
  } else {
    fmt.Fprint(w, strings.TrimSpace(res), "\n")
  }
}

func main() {
  http.HandleFunc("/ping/", ping)
  http.HandleFunc("/mtr/", mtr)
  http.HandleFunc("/tracert/", traceroute)
  logstdout.Info("Serving on :", listenport)
  logfile.Info("Serving on :", listenport)
  http.ListenAndServe(fmt.Sprint(":", listenport), nil)
}