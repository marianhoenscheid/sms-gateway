package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	configTemplate = "/etc/gammu-smsdrc.template"
	configPath     = "/tmp/gammu-smsdrc"
)

var candidates = []string{"/dev/ttyUSB0", "/dev/ttyUSB1", "/dev/ttyUSB2"}

func detectATPort() (string, error) {
	tmp, err := os.CreateTemp("", "gammu-identify-*.conf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	for _, dev := range candidates {
		if _, err := os.Stat(dev); err != nil {
			continue
		}
		cfg := fmt.Sprintf("[gammu]\ndevice = %s\nconnection = at\n", dev)
		if err := os.WriteFile(tmp.Name(), []byte(cfg), 0o600); err != nil {
			return "", err
		}
		// #nosec G204 -- gammu and --identify are hardcoded; only the temp config path varies.
		cmd := exec.Command("gammu", "-c", tmp.Name(), "--identify")
		if err := cmd.Run(); err == nil {
			return dev, nil
		}
	}
	return "", fmt.Errorf("no AT port found")
}

func writeConfig(device string) error {
	data, err := os.ReadFile(configTemplate)
	if err != nil {
		return err
	}
	out := strings.ReplaceAll(string(data), "device = /dev/ttyUSB0", "device = "+device)
	// #nosec G703 G306 -- configPath is a hardcoded /tmp path; device value is validated against /dev/ttyUSB* candidates.
	return os.WriteFile(configPath, []byte(out), 0o600)
}

func main() {
	for {
		fmt.Println("entrypoint: suche AT-Port...")
		port, err := detectATPort()
		if err != nil {
			fmt.Println("entrypoint: kein AT-Port gefunden, retry in 15s")
			time.Sleep(15 * time.Second)
			continue
		}
		fmt.Println("entrypoint: nutze", port)
		if err := writeConfig(port); err != nil {
			fmt.Fprintf(os.Stderr, "entrypoint: config write failed: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// #nosec G204 -- gammu-smsd and -c are hardcoded; only the config path varies.
		cmd := exec.Command("gammu-smsd", "-c", configPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "entrypoint: gammu-smsd exited: %v\n", err)
		}
		fmt.Println("entrypoint: gammu-smsd beendet, re-detect in 5s")
		time.Sleep(5 * time.Second)
	}
}

func init() {
	// Ensure spool directories exist.
	for _, d := range []string{"inbox", "outbox", "sent", "error"} {
		// #nosec G301 -- spool directories need group/other read+execute for gammu-smsd file service.
		if err := os.MkdirAll(filepath.Join("/var/spool/gammu", d), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "entrypoint: failed to create spool dir %s: %v\n", d, err)
		}
	}
}
