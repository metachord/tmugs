package main

import (
	"code.google.com/p/gopass"
	"flag"
	"fmt"
	"github.com/moraes/config"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"tmux"
)

func init() {
}

type tmugsCfg struct {
	*config.Config
	*tmux.Session
	name string
	root string
}

func main() {
	flag.Parse()
	args := flag.Args()
	processArgs(args)
}

func usage() {
	msg := []string{
		"usage: %s",
		" start CFG1 [CFG2 CFG3 ...] - start sessions with configurations",
		" ls - list sessions",
		" ls CFG - list windows in session",
		" a CFG WINDOW - attach to specified window in session",
		" kill CFG1 [CFG2 CFG3 ...] - kill specified sessions",
		"\n",
	}
	fmt.Fprintf(os.Stderr, strings.Join(msg, "\n"), os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func processArgs(args []string) {
	if len(args) == 0 {
		usage()
	}
	switch args[0] {
	case "start":
		if len(args) > 1 {
			startSessions(args[1:])
		} else {
			usage()
		}
	case "ls":
		switch len(args) {
		case 1:
			listSessions()
		case 2:
			listWindows(args[1])
		default:
			usage()
		}
	case "a":
		if len(args) == 3 {
			attachWindow(args[1], args[2])
		} else {
			usage()
		}
	case "kill":
		if len(args) > 1 {
			killSessions(args[1:])
		} else {
			usage()
		}
	default:
		usage()
	}
}

func startSessions(sessions []string) {
	for _, filename := range sessions {
		c := getCfg(filename)
		c.processTabs()
	}
}

func killSessions(sessions []string) {
	for _, sessionname := range sessions {
		tmux.KillSession(sessionname)
	}
}

func listSessions() {
	sessions, _ := tmux.ListSessions()
	fmt.Printf("%v\n", strings.Join(sessions, "\n"))
	return
}

func listWindows(session string) {
	windows, _ := tmux.ListWindows(session)
	fmt.Printf("%v\n", strings.Join(windows, "\n"))
	return
}

func attachWindow(session string, window string) {
	tmux.SelectWindow(session, window)
}

func getCfg(filename string) (c *tmugsCfg) {
	cfg, err := config.ParseYamlFile(filename)
	if err != nil {
		log.Panic(err)
	}

	bn := filepath.Base(filename)
	pn := strings.Split(bn, ".")[0]

	proot, err := cfg.String("root")
	if err != nil {
		proot, err = os.Getwd()
		if err != nil {
			proot = "~/"
		}
	}

	_, err = cfg.String("sudo")
	if err != nil {
		log.Printf("No SUDO")
	} else {
		log.Printf("SUDO")
		getSudoPass()
	}

	ts, err := tmux.NewSession(pn)
	if err != nil {
		log.Fatal(err)
	}
	c = &tmugsCfg{
		Config:  cfg,
		Session: ts,
		name:    pn,
		root:    proot,
	}
	return
}

var SUDOPASS string

func getSudoPass() {
	isValid := false
	for !isValid {
		pass, err := gopass.GetPass("Enter SUDO password: ")
		if err != nil {
			log.Panic(err)
		}
		cmd := exec.Command("sudo", "-k", "-S", "whoami")
		if err != nil {
			log.Panic(err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Panic(err)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Panic(err)
		}

		err = cmd.Start()
		if err != nil {
			log.Panic(err)
		}
		n, err := io.WriteString(stdin, pass+"\n")

		if err != nil {
			log.Panic(err)
		} else {
			log.Printf("Write %d bytes", n)
		}

		outB, err := ioutil.ReadAll(stdout)

		if err != nil {
			log.Panic(err)
		}

		log.Printf("Out: %#v", string(outB))
		err = cmd.Wait()

		if err != nil {
			log.Panic(err)
		}

		SUDOPASS = pass
		isValid = true
	}
}

func (c *tmugsCfg) tabs() (ts []interface{}) {
	ts, err := c.List("tabs")
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Tabs: %#v", ts)
	return
}

func (c *tmugsCfg) processTabs() {
	ts, err := c.List("tabs")
	if err != nil {
		log.Panic(err)
	}
	for _, tab := range ts {
		c.newWindow(tab.(map[string]interface{}))
	}
}

func sleep(s int) {
	time.Sleep(time.Duration(s) * time.Second)
}

func (c *tmugsCfg) newWindow(tab map[string]interface{}) {
	log.Printf("Tab: %#v", tab)
	if 1 != len(tab) {
		log.Printf("ERROR: Bad tab: %#v", tab)
		return
	}
	for k, v := range tab {
		sleepS, ok := v.(map[string]interface{})["sleep"].(int)
		if !ok {
			sleepS = 0
		}
		sleep(sleepS)
		cd, ok := v.(map[string]interface{})["cd"].(string)
		if !ok {
			cd = "."
		}
		dir := filepath.Join(c.root, cd)
		_, err := c.NewWindow(k, dir)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		command, ok := v.(map[string]interface{})["run"]
		if ok {
			switch command := command.(type) {
			case string:
				log.Printf("Exec: %s", command)
				c.SendCommand(command)
			case []interface{}:
				for _, rv := range command {
					switch rv := rv.(type) {
					case string:
						log.Printf("Exec: %s", rv)
						c.SendCommand(rv)
					case map[string]interface{}:
						if len(rv) == 1 {
							// Take name of the key
							var rkey string
							for rkey, _ = range rv {
								break
							}

							switch rkey {
							case "sudo":
								log.Printf("Run with SUDO: %#v", rv[rkey])
								c.SendCommand("sudo " + rv[rkey].(string))
								c.SendCommand(SUDOPASS)
							default:
								log.Printf("ERR: Unknown type of run: %#v", rkey)
							}
						} else {
							log.Printf("ERR: bad command: %#v", rv)
						}
					default:
						log.Printf("ERR: Bad command: %#v", rv)
					}
					sleep(sleepS)
				}
			default:
				log.Printf("ERR: Unknown type of run: %#v", command)
			}
		}
	}
}
