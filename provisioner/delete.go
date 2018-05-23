package main

import (
	"errors"
	"fmt"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

func (p *xenServerProvisioner) Delete(volume *v1.PersistentVolume) error {
	glog.Infof("Delete called for volume: %s", volume.Name)

	if err := p.deleteFromXenServer(volume.ObjectMeta.Name); err != nil {
		glog.Errorf("Failed to delete volume %s, error: %s", volume, err.Error())
		return err
	}

	return nil
}

func (p *xenServerProvisioner) deleteFromXenServer(nameLabel string) error {
	xapi, session, err := p.xapiLogin()
	if err != nil {
		return errors.New(fmt.Sprintf("Could not login at XenServer, error: %s", err.Error()))
	}
	defer func() {
		if err := p.xapiLogout(xapi, session); err != nil {
			glog.Errorf("Failed to log out from XenServer, error: %s", err.Error())
		}
	}()

	vdis, err := xapi.VDI.GetByNameLabel(session, nameLabel)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not list VDIs for name label %s, error: %s", nameLabel, err.Error()))
	}

	if len(vdis) > 1 {
		return errors.New(fmt.Sprintf("Too many VDIs where found for name label %s", nameLabel))
	}

	if len(vdis) > 0 {
		err := xapi.VDI.Destroy(session, vdis[0])
		if err != nil {
			return errors.New(fmt.Sprintf("Could not destroy VDI for name label %s, error: %s", nameLabel, err.Error()))
		}

		glog.Infof("VDI was destroyed for name label %s", nameLabel)
	} else {
		glog.Infof("VDI was already destroyed for name label %s", nameLabel)
	}

	return nil
}
