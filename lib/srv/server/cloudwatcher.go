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
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/cloudflare/cfssl/log"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/cloud"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/srv/db/common"
	"github.com/gravitational/trace"
)

type EC2Instances struct {
	Region    string
	Document  string
	Instances []*ec2.Instance
}

type Watcher struct {
	// Instances can be used to consume
	Instances chan EC2Instances

	fetchers []*ec2InstanceFetcher
	waitTime time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

func (w *Watcher) Run() {
	ticker := time.NewTicker(w.waitTime)
	for {
		for _, fetcher := range w.fetchers {
			inst, err := fetcher.GetEC2Instances(w.ctx)
			if err != nil {
				log.Error("Failed to fetch EC2 instances: ", err)
				continue
			}
			w.Instances <- *inst
		}
		select {
		case <-ticker.C:
			continue
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Watcher) Stop() {
	w.cancel()
}

func NewCloudServerWatcher(ctx context.Context, matchers []services.AWSMatcher, clients cloud.Clients) (*Watcher, error) {
	cancelCtx, cancelFn := context.WithCancel(ctx)
	watcher := Watcher{
		fetchers:  []*ec2InstanceFetcher{},
		ctx:       cancelCtx,
		cancel:    cancelFn,
		waitTime:  time.Minute,
		Instances: make(chan EC2Instances),
	}
	for _, matcher := range matchers {
		for _, region := range matcher.Regions {
			cl, err := clients.GetAWSEC2Client(region)
			if err != nil {
				return nil, trace.Wrap(err)
			}
			fetcher :=
				newEc2InstanceFetcher(matcher, region, matcher.SSMDocument, cl, matcher.Tags)
			if err != nil {
				return nil, trace.Wrap(err)
			}
			watcher.fetchers = append(watcher.fetchers, fetcher)
		}
	}
	return &watcher, nil
}

type ec2InstanceFetcher struct {
	Filters  []*ec2.Filter
	EC2      ec2iface.EC2API
	Region   string
	Document string
}

func newEc2InstanceFetcher(matcher services.AWSMatcher, region, document string,
	ec2Client ec2iface.EC2API, labels types.Labels) *ec2InstanceFetcher {
	tagFilters := make([]*ec2.Filter, 0, len(labels)+1)
	tagFilters = append(tagFilters, &ec2.Filter{
		Name:   aws.String("instance-state-name"),
		Values: aws.StringSlice([]string{ec2.InstanceStateNameRunning}),
	})
	for key, val := range labels {
		tagFilters = append(tagFilters, &ec2.Filter{
			Name:   aws.String("tag:" + key),
			Values: aws.StringSlice(val),
		})
	}
	fetcherConfig := ec2InstanceFetcher{
		EC2:      ec2Client,
		Filters:  tagFilters,
		Region:   region,
		Document: document,
	}
	return &fetcherConfig
}

func (f *ec2InstanceFetcher) GetEC2Instances(ctx context.Context) (*EC2Instances, error) {
	var instances []*ec2.Instance
	err := f.EC2.DescribeInstancesPagesWithContext(ctx, &ec2.DescribeInstancesInput{
		Filters: f.Filters,
	},
		func(dio *ec2.DescribeInstancesOutput, b bool) bool {
			for _, res := range dio.Reservations {
				instances = append(instances, res.Instances...)
			}
			return true
		})

	if err != nil {
		return nil, common.ConvertError(err)
	}

	return &EC2Instances{
		Region:    f.Region,
		Document:  f.Document,
		Instances: instances,
	}, nil
}
