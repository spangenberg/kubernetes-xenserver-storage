package main

import (
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	xenapi "github.com/ringods/go-xen-api-client"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultFSType = "ext4"

func (p *xenServerProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	glog.Infof("Provision called for volume: %s", options.PVName)

	if err := p.provisionOnXenServer(options); err != nil {
		glog.Errorf("Failed to provision volume %s, error: %s", options, err.Error())
		return nil, err
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
		},
		Spec: v1.PersistentVolumeSpec{
			AccessModes: options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver: driver,
					FSType: defaultFSType,
					Options: map[string]string{
						driverOptionXenServerHost:     p.xenServerHost,
						driverOptionXenServerUsername: p.xenServerUsername,
						driverOptionXenServerPassword: p.xenServerPassword,
					},
				},
			},
		},
	}

	return pv, nil
}

func (p *xenServerProvisioner) provisionOnXenServer(options controller.VolumeOptions) error {
	xapi, session, err := p.xapiLogin()
	if err != nil {
		return fmt.Errorf("Could not login at XenServer, error: %s", err.Error())
	}
	defer func() {
		if err := p.xapiLogout(xapi, session); err != nil {
			glog.Errorf("Failed to log out from XenServer, error: %s", err.Error())
		}
	}()

	srNameLabel := options.Parameters[storageClassParameterSRName]
	srs, err := xapi.SR.GetByNameLabel(session, srNameLabel)
	if err != nil {
		return fmt.Errorf("Could not list SRs for name label %s, error: %s", srNameLabel, err.Error())
	}

	if len(srs) > 1 {
		return fmt.Errorf("Too many SRs where found for name label %s", srNameLabel)
	}

	if len(srs) < 1 {
		return fmt.Errorf("No SR was found for name label %s", srNameLabel)
	}

	capacity, exists := options.PVC.Spec.Resources.Requests[v1.ResourceStorage]
	if !exists {
		return fmt.Errorf("Capacity was not specified for name label %s", options.PVName)
	}

	_, err = xapi.VDI.Create(session, xenapi.VDIRecord{
		NameDescription: "Kubernetes Persisted Volume Claim",
		NameLabel:       options.PVName,
		SR:              srs[0],
		Type:            xenapi.VdiTypeUser,
		VirtualSize:     int(capacity.Value()),
	})
	if err != nil {
		return fmt.Errorf("Could not create VDI for name label %s, error: %s", options.PVName, err.Error())
	}

	return nil
}
