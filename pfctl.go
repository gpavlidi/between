package between

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"text/template"
)

const pfctlConfigTemplate = `echo '
Interface = {{.NetIf}} #en0
Proxy_User = root
Proxy_IP = 127.0.0.1
Proxy_Http_Port = {{.HttpPort}} #8000
Proxy_Https_Port = {{.HttpsPort}} #8001

# Intercept http/https packets
Filter = "proto tcp from " $Interface " to any port { 80, 443 }"
Http_Filter = "proto tcp from " $Interface " to any port 80"
Https_Filter = "proto tcp from " $Interface " to any port 443"

# Forward intercepted packets to our Proxy
rdr pass log on lo0 $Http_Filter -> $Proxy_IP port $Proxy_Http_Port
rdr pass log on lo0 $Https_Filter -> $Proxy_IP port $Proxy_Https_Port

# Route outgoing packets to loopback so I can filter them (only incoming are filterable)
# Skip requests of proxy to avoid infinite loop
pass out on $Interface route-to lo0 inet $Filter user != $Proxy_User keep state
' | sudo pfctl -f -`

func Enable() error {
	fmt.Println("INFO:", "In case connectivity is lost after program exits run:", "sudo pfctl -d")
	out, err := run("sudo pfctl -e")
	if err == nil || (strings.Contains(out, "pf enabled") || strings.Contains(out, "pf already enabled")) {
		return nil
	}
	return errors.New("Cannot enable pf:\n" + out)
}

func Disable() error {
	out, err := run("sudo pfctl -d")
	if err == nil || (strings.Contains(out, "pf disabled") || strings.Contains(out, "pf not enabled")) {
		return nil
	}
	return errors.New("Cannot disable pf:\n" + out)
}

func Configure(httpPort, httpsPort, netIf string) error {
	var pfctlConfig bytes.Buffer

	t := template.Must(template.New("pfctlConfigTemplate").Parse(pfctlConfigTemplate))

	err := t.Execute(&pfctlConfig,
		struct{ HttpPort, HttpsPort, NetIf string }{HttpPort: httpPort, HttpsPort: httpsPort, NetIf: netIf})
	if err != nil {
		return err
	}

	out, err := run(pfctlConfig.String())
	if err != nil || strings.Contains(out, "pf rules not loaded") || strings.Contains(out, "Permission denied") {
		return errors.New("Cannot configure pf:\n" + out)
	}
	return nil
}

func CheckCompatibility() error {
	if !isRoot() {
		return errors.New("This needs to be run as root.")
	}
	if !isOSX() {
		return errors.New("Currently only OSX is supported.")
	}
	if !hasPfctl() {
		return errors.New("Could not find pfctl in PATH.")
	}
	return nil
}

func run(cmd string) (string, error) {
	output, err := exec.Command("/bin/sh", "-c", cmd+" 2>&1").Output()
	return string(output), err
}

func which(file string) (string, error) {
	return exec.LookPath(file)
}

func isOSX() bool {
	return "darwin" == runtime.GOOS
}

func isRoot() bool {
	usr, _ := user.Current()
	return "root" == usr.Username
}

func hasPfctl() bool {
	_, err := which("pfctl")
	return err == nil
}
