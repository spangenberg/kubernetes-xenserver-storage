package main

import (
	"fmt"
	"os"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	xenapi "github.com/ringods/go-xen-api-client"
	"k8s.io/utils/exec"
)

type xenServerProvisioner struct {
	runner            exec.Interface
	xenServerHost     string
	xenServerUsername string
	xenServerPassword string
}

func NewXenServerProvisioner() controller.Provisioner {
	return &xenServerProvisioner{
		runner:            exec.New(),
		xenServerHost:     os.Getenv("XENSERVER_HOST"),
		xenServerUsername: os.Getenv("XENSERVER_USERNAME"),
		xenServerPassword: os.Getenv("XENSERVER_PASSWORD"),
	}
}

func (p *xenServerProvisioner) xapiLogin() (*xenapi.Client, xenapi.SessionRef, error) {
	xapi, err := xenapi.NewClient(fmt.Sprintf("https://%s", p.xenServerHost), nil)
	if err != nil {
		return nil, "", err
	}

	session, err := xapi.Session.LoginWithPassword(p.xenServerUsername, p.xenServerPassword, "1.0", *provisioner)
	if err != nil {
		return nil, "", err
	}

	return xapi, session, nil
}

func (p *xenServerProvisioner) xapiLogout(xapi *xenapi.Client, session xenapi.SessionRef) error {
	return xapi.Session.Logout(session)
}
