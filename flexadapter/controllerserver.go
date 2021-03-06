/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubernetes-csi/drivers/lib"
)

const (
	deviceID = "deviceID"
	fsType   = "fsType"
)

type controllerServer struct {
	lib.ControllerServerDefaults
	driver *lib.CSIDriver
}

func GetVersionString(ver *csi.Version) string {
	return fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Patch)
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	if err := cs.driver.ValidateRequest(req.Version, csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		return nil, err
	}

	call := adapter.flexDriver.NewDriverCall(attachCmd)
	fsType := req.GetVolumeCapability().GetMount().FsType
	call.AppendSpec(req.GetVolumeId(), fsType, req.GetReadonly(), req.GetVolumeAttributes())
	call.Append(req.GetNodeId())

	callStatus, err := call.Run()
	if isCmdNotSupportedErr(err) {
		return nil, status.Error(codes.Unimplemented, "")
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pvInfo := map[string]string{}

	pvInfo[deviceID] = callStatus.DevicePath

	return &csi.ControllerPublishVolumeResponse{
		PublishVolumeInfo: pvInfo,
	}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	if err := cs.Driver.ValidateRequest(req.Version, csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME); err != nil {
		return nil, err
	}

	call := adapter.flexDriver.NewDriverCall(detachCmd)
	call.Append(req.GetVolumeId())
	call.Append(req.GetNodeId())

	_, err := call.Run()
	if isCmdNotSupportedErr(err) {
		return nil, status.Error(codes.Unimplemented, "")
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	for _, cap := range req.VolumeCapabilities {
		if cap.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			return &csi.ValidateVolumeCapabilitiesResponse{false, ""}, nil
		}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{true, ""}, nil
}
