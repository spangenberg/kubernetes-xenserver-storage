package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	xenapi "github.com/ringods/go-xen-api-client"
)

const (
	defaultMountFlags   = syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	defaultUnmountFlags = syscall.MNT_DETACH
	debugLogFile        = "/tmp/xenserver-driver.log"
)

type jsonParameter struct {
	FSGroup           string `json:"kubernetes.io/fsGroup"`
	FSType            string `json:"kubernetes.io/fsType"`
	PVOrVolumeName    string `json:"kubernetes.io/pvOrVolumeName"`
	PodName           string `json:"kubernetes.io/pod.name"`
	PodNamespace      string `json:"kubernetes.io/pod.namespace"`
	PodUID            string `json:"kubernetes.io/pod.uid"`
	ReadWrite         string `json:"kubernetes.io/readwrite"`
	ServiceAccount    string `json:"kubernetes.io/serviceAccount.name"`
	XenServerHost     string `json:"spangenberg.io/xenserver/host"`
	XenServerPassword string `json:"spangenberg.io/xenserver/password"`
	XenServerUsername string `json:"spangenberg.io/xenserver/username"`
}

func main() {
	var command string
	var mountDir string
	var jsonOptions string

	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if len(os.Args) > 2 {
		mountDir = os.Args[2]
	}
	if len(os.Args) > 3 {
		jsonOptions = os.Args[3]
	}

	debug(fmt.Sprintf("%s %s %s", command, mountDir, jsonOptions))

	switch command {
	case "init":
		fmt.Print("{\"status\": \"Success\", \"capabilities\": {\"attach\": false}}")
		os.Exit(0)
	case "mount":
		mount(mountDir, jsonOptions)
	case "unmount":
		unmount(mountDir)
	default:
		fmt.Print("{\"status\": \"Not supported\"}")
		os.Exit(1)
	}
}

func debug(message string) {
	if _, err := os.Stat(debugLogFile); err == nil {
		f, _ := os.OpenFile(debugLogFile, os.O_APPEND|os.O_WRONLY, 0600)
		defer f.Close()
		f.WriteString(fmt.Sprintln(message))
	}
}

func success() {
	debug("SUCCESS")

	fmt.Print("{\"status\": \"Success\"}")

	os.Exit(0)
}

func failure(err error) {
	debug(fmt.Sprintf("FAILURE - %s", err.Error()))

	failureMap := map[string]string{"status": "Failure", "message": err.Error()}
	jsonMessage, _ := json.Marshal(failureMap)
	fmt.Print(string(jsonMessage))

	os.Exit(1)
}

func mount(mountDir, jsonOptions string) {
	byt := []byte(jsonOptions)
	options := jsonParameter{}
	if err := json.Unmarshal(byt, &options); err != nil {
		failure(err)
	}

	if options.FSType == "" {
		options.FSType = "ext4"
	}

	flags := defaultMountFlags
	var mode xenapi.VbdMode
	switch options.ReadWrite {
	case "ro":
		flags = flags | syscall.MS_RDONLY
		mode = xenapi.VbdModeRO
	case "rw":
		mode = xenapi.VbdModeRW
	default:
		failure(errors.New("Unknown ReadWrite"))
	}

	xapi, session, err := xapiLogin(options.XenServerHost, options.XenServerUsername, options.XenServerPassword)
	if err != nil {
		failure(fmt.Errorf("Could not login at XenServer, error: %s", err.Error()))
	}
	defer func() {
		if err := xapiLogout(xapi, session); err != nil {
			failure(fmt.Errorf("Failed to log out from XenServer, error: %s", err.Error()))
		}
	}()

	vm, err := getVM(xapi, session)
	if err != nil {
		failure(err)
	}

	debug("VM.GetAllowedVBDDevices")
	vbdDevices, err := xapi.VM.GetAllowedVBDDevices(session, vm)
	if err != nil {
		failure(err)
	}

	if len(vbdDevices) < 1 {
		failure(errors.New("No VBD devices are available anymore"))
	}

	debug("VDI.GetAllRecords")
	vdis, err := xapi.VDI.GetAllRecords(session)
	if err != nil {
		failure(err)
	}

	var vdiUUID xenapi.VDIRef
	for ref, vdi := range vdis {
		if vdi.NameLabel == options.PVOrVolumeName && !vdi.IsASnapshot {
			vdiUUID = ref
		}
	}
	if vdiUUID == "" {
		failure(errors.New("Could not find VDI"))
	}

	debug("VBD.GetAllRecords")
	vbds, err := xapi.VBD.GetAllRecords(session)
	if err != nil {
		failure(err)
	}

	for ref, vbd := range vbds {
		if vbd.VDI == vdiUUID && vbd.CurrentlyAttached {
			if err := detachVBD(ref, xapi, session); err != nil {
				failure(err)
			}
		}
	}

	debug("VBD.Create")
	vbdUUID, err := xapi.VBD.Create(session, xenapi.VBDRecord{
		Bootable:    false,
		Mode:        mode,
		Type:        xenapi.VbdTypeDisk,
		Unpluggable: true,
		Userdevice:  vbdDevices[0],
		VDI:         vdiUUID,
		VM:          vm,
	})
	if err != nil {
		failure(err)
	}

	debug("VBD.Plug")
	if err := xapi.VBD.Plug(session, vbdUUID); err != nil {
		failure(err)
	}

	debug("VBD.GetDevice")
	device, err := xapi.VBD.GetDevice(session, vbdUUID)
	if err != nil {
		failure(err)
	}
	devicePath := fmt.Sprintf("/dev/%s", device)

	blkid, err := run("blkid", devicePath)
	if err != nil && !strings.Contains(err.Error(), "exit status 2") {
		failure(err)
	}

	if blkid == "" {
		if _, err := run("mkfs", "-t", options.FSType, devicePath); err != nil {
			failure(err)
		}
	}

	debug("ioutil.WriteFile")
	if err := ioutil.WriteFile(fmt.Sprintf("%s-json", mountDir), byt, 0600); err != nil {
		failure(err)
	}

	debug("os.MkdirAll")
	if err := os.MkdirAll(mountDir, 0755); err != nil {
		failure(err)
	}

	debug("syscall.Mount")
	if err := syscall.Mount(devicePath, mountDir, options.FSType, uintptr(flags), ""); err != nil {
		failure(err)
	}

	success()
}

func unmount(mountDir string) {
	byt, err := ioutil.ReadFile(fmt.Sprintf("%s-json", mountDir))
	if err != nil {
		failure(err)
	}

	options := jsonParameter{}
	if err := json.Unmarshal(byt, &options); err != nil {
		failure(err)
	}

	xapi, session, err := xapiLogin(options.XenServerHost, options.XenServerUsername, options.XenServerPassword)
	if err != nil {
		failure(fmt.Errorf("Could not login at XenServer, error: %s", err.Error()))
	}
	defer func() {
		if err := xapiLogout(xapi, session); err != nil {
			failure(fmt.Errorf("Failed to log out from XenServer, error: %s", err.Error()))
		}
	}()

	vm, err := getVM(xapi, session)
	if err != nil {
		failure(err)
	}

	devicePath, err := run("findmnt", "-n", "-o", "SOURCE", "--target", mountDir)
	if err != nil {
		failure(err)
	}

	devicePathElements := strings.Split(devicePath, "/")
	if len(devicePathElements) < 3 || len(devicePathElements) > 3 {
		failure(errors.New("Device path is incorrect"))
	}

	device := devicePathElements[2]

	debug("syscall.Unmount")
	if err := syscall.Unmount(mountDir, defaultUnmountFlags); err != nil {
		failure(err)
	}

	debug("VBD.GetAllRecords")
	vbds, err := xapi.VBD.GetAllRecords(session)
	if err != nil {
		failure(err)
	}

	for ref, vbd := range vbds {
		if vbd.VM == vm && vbd.Device == device && vbd.CurrentlyAttached {
			if err := detachVBD(ref, xapi, session); err != nil {
				failure(err)
			}
		}
	}

	debug("os.Remove")
	if err := os.Remove(fmt.Sprintf("%s-json", mountDir)); err != nil {
		failure(err)
	}

	success()
}

func detachVBD(vbd xenapi.VBDRef, xapi *xenapi.Client, session xenapi.SessionRef) error {
	debug("VBD.Unplug")
	if err := xapi.VBD.Unplug(session, vbd); err != nil {
		if err != nil && !strings.Contains(err.Error(), xenapi.ERR_DEVICE_DETACH_REJECTED) {
			return err
		}

		debug("VBD.UnplugForce")
		if err := xapi.VBD.UnplugForce(session, vbd); err != nil {
			return err
		}
	}

	debug("VBD.Destroy")
	return xapi.VBD.Destroy(session, vbd)
}

func getMAC() (string, error) {
	debug("net.Interfaces")
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var mac string
	for _, i := range interfaces {
		if i.Name == "eth0" && i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
			mac = i.HardwareAddr.String()
		}
	}

	if mac == "" {
		return "", errors.New("MAC address not found")
	}

	return mac, nil
}

func getVM(xapi *xenapi.Client, session xenapi.SessionRef) (xenapi.VMRef, error) {
	mac, err := getMAC()
	if err != nil {
		return "", err
	}

	debug("VIF.GetAllRecords")
	vifs, err := xapi.VIF.GetAllRecords(session)
	if err != nil {
		return "", err
	}

	var vm xenapi.VMRef
	for _, vif := range vifs {
		if vif.MAC == mac && vif.CurrentlyAttached {
			vm = vif.VM
		}
	}

	if vm == "" {
		return "", errors.New("Could not find VM with MAC")
	}

	return vm, nil
}

func run(cmd string, args ...string) (string, error) {
	debug(fmt.Sprintf("Running %s %s", cmd, args))
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Error running %s %v: %v, %s", cmd, args, err, out)
	}
	return string(out), nil
}

func xapiLogin(host, username, password string) (*xenapi.Client, xenapi.SessionRef, error) {
	xapi, err := xenapi.NewClient(fmt.Sprintf("https://%s", host), nil)
	if err != nil {
		return nil, "", err
	}

	session, err := xapi.Session.LoginWithPassword(username, password, "1.0", "spangenberg.io/xenserver")
	if err != nil {
		return nil, "", err
	}

	return xapi, session, nil
}

func xapiLogout(xapi *xenapi.Client, session xenapi.SessionRef) error {
	return xapi.Session.Logout(session)
}
