package tmux

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Session struct {
	name string
}

type TmuxError struct {
	Err    string
	Reason string
}

func (e TmuxError) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.Reason)
}

func CreateSession(name string) (ts *Session) {
	ts = new(Session)
	ts.name = name
	return
}

func NewSession(name string) (ts *Session, err error) {
	out, err := execTmux("new-session", "-d", "-s", name)
	if err != nil {
		log.Printf("ERR: %v", err)
		return nil, TmuxError{err.Error(), out}
	}
	return CreateSession(name), nil
}

func (ts *Session) NewWindow(name string, dir string) (out string, err error) {
	log.Printf("Run new window %s in dir %v", name, dir)
	out, err = execTmux("new-window",
		"-P", "-F", "#{session_name}:#{window_index}",
		"-t", ts.name,
		"-c", dir,
		"-n", name)
	return
}

func (ts *Session) SendCommand(command string) (out string, err error) {
	out, err = execTmux("send-keys", "-t", ts.name, command, "Enter")
	return
}

func ListSessions() (sessions []string, err error) {
	out, err := execTmux("list-sessions", "-F", "#{session_name}")
	sessions = strings.Split(out, "\n")
	sessions = sessions[:len(sessions)-1]
	return
}

func ListWindowsIdx(session string) (windows []string, err error) {
	out, err := execTmux("list-windows",
		"-t", session,
		"-F", "#{window_index}:#{window_name}")
	windows = strings.Split(out, "\n")
	windows = windows[:len(windows)-1]
	return
}

func ListWindows(session string) (windows []string, err error) {
	out, err := execTmux("list-windows",
		"-t", session,
		"-F", "#{window_name}")
	windows = strings.Split(out, "\n")
	windows = windows[:len(windows)-1]
	return
}

func SelectWindow(session string, window string) {
	out, err := execTmux("select-window", "-t", fmt.Sprintf("%s:%s", session, window))
	if err != nil {
		fmt.Printf("%s: %s", err, out)
	}
}

func KillSession(session string) (out string, err error) {
	out, err = execTmux("kill-session", "-t", session)
	return
}

func execTmux(params ...string) (out string, err error) {
	outb, err := exec.Command("tmux", params...).CombinedOutput()
	out = string(outb)
	return
}
