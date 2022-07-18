/*
Copyright 2022 Gravitational, Inc.

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

package server

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/gravitational/teleport/api/types/events"
	libevent "github.com/gravitational/teleport/lib/events"
	"github.com/gravitational/trace"
)

type Installation struct {
	instances []*string
	SSM       ssmiface.SSMAPI
	rechecker time.Ticker
	params    map[string][]*string
}

func NewInstallation(client ssmiface.SSMAPI, instances []*ec2.Instance, params map[string]string) *Installation {
	var ids []*string

	for _, inst := range instances {
		ids = append(ids, inst.InstanceId)
	}

	ssmParams := make(map[string][]*string)

	for key, val := range params {
		ssmParams[key] = []*string{aws.String(val)}
	}

	return &Installation{
		instances: ids,
		SSM:       client,
		rechecker: *time.NewTicker(time.Second * 30),
		params:    ssmParams,
	}
}

var ErrCommandInProgress = errors.New("command in progress")

func (i *Installation) checkCommands(commandID *string) ([]*events.EC2DiscoveryScriptRun, error) {
	var resultCmds []*events.EC2DiscoveryScriptRun
	for _, inst := range i.instances {
		cmdOut, err := i.SSM.GetCommandInvocation(&ssm.GetCommandInvocationInput{
			CommandId:  commandID,
			InstanceId: inst,
		})
		if err != nil {
			return nil, trace.Wrap(err)
		}
		status := aws.StringValue(cmdOut.Status)
		if status == ssm.CommandStatusInProgress {
			return nil, trace.Wrap(ErrCommandInProgress)
		}

		var code string
		if status == ssm.CommandStatusFailed {
			code = libevent.DiscoveryScriptEC2FailCode
		} else {
			code = libevent.DiscoveryScriptEC2SuccessCode
		}

		event := events.EC2DiscoveryScriptRun{
			Metadata: events.Metadata{
				Type: libevent.EC2DiscoveryInstallScriptEvent,
				Code: code,
			},
			CommandID:  aws.StringValue(commandID),
			InstanceID: aws.StringValue(inst),
			ExitCode:   aws.Int64Value(cmdOut.ResponseCode),
			Status:     status,
		}

		resultCmds = append(resultCmds, &event)
	}
	return resultCmds, nil
}

func (i *Installation) DoInstall(document string) ([]*events.EC2DiscoveryScriptRun, error) {
	output, err := i.SSM.SendCommand(&ssm.SendCommandInput{
		DocumentName: aws.String(document),
		InstanceIds:  i.instances,
		Parameters:   i.params,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	commandID := output.Command.CommandId
	for {
		<-i.rechecker.C
		result, err := i.checkCommands(commandID)
		if err != nil {
			if errors.Is(err, ErrCommandInProgress) {
				continue
			}
			return result, err
		}
		return result, nil
	}
}
