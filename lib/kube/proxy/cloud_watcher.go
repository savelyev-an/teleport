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

package proxy

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/cloudflare/cfssl/log"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/srv/db/common"
	"github.com/gravitational/trace"
)

type Watcher struct {
	// Instances can be used to consume
	Instances chan []*eks.Cluster

	fetchers []fetcher
	waitTime time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

func (w *Watcher) Start() {
	ticker := time.NewTicker(w.waitTime)
	for {
		for _, fetcher := range w.fetchers {
			inst, err := fetcher.GetKubeClusters(w.ctx)
			if err != nil {
				log.Error("Failed to fetch EC2 instances: ", err)
				continue
			}
			w.Instances <- inst
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

func NewCloudServerWatcher(ctx context.Context, matchers []services.AWSMatcher, clients common.CloudClients) (*Watcher, error) {
	cancelCtx, cancelFn := context.WithCancel(ctx)
	watcher := Watcher{
		fetchers:  []fetcher{},
		ctx:       cancelCtx,
		cancel:    cancelFn,
		waitTime:  time.Minute,
		Instances: make(chan []*eks.Cluster),
	}
	for _, matcher := range matchers {
		for _, region := range matcher.Regions {
			cl, err := clients.GetAWSEKSClient(region)
			if err != nil {
				return nil, trace.Wrap(err)
			}
			fetcher, err := newEc2InstanceFetcher(matcher, region, cl)
			if err != nil {
				return nil, trace.Wrap(err)
			}
			watcher.fetchers = append(watcher.fetchers, fetcher)
		}
	}
	return &watcher, nil
}

type fetcher interface {
	GetKubeClusters(context.Context) ([]*eks.Cluster, error)
}

type ec2InstanceFetcher struct {
	Labels types.Labels
	EKS    eksiface.EKSAPI
	Region string
}

func newEc2InstanceFetcher(matcher services.AWSMatcher, region string, eksClient eksiface.EKSAPI) (*ec2InstanceFetcher, error) {
	fetcherConfig := ec2InstanceFetcher{
		EKS:    eksClient,
		Labels: matcher.Tags,
		Region: region,
	}
	return &fetcherConfig, nil
}

func (f *ec2InstanceFetcher) GetKubeClusters(ctx context.Context) ([]*eks.Cluster, error) {
	var (
		clusters []*eks.Cluster
		mu       sync.Mutex
	)
	err := f.EKS.ListClustersPagesWithContext(ctx,
		&eks.ListClustersInput{},
		func(lCusters *eks.ListClustersOutput, _ bool) bool {
			wg := &sync.WaitGroup{}
			wg.Add(len(lCusters.Clusters))
			for i := 0; i < len(lCusters.Clusters); i++ {
				eksClusterName := lCusters.Clusters[i]

				go func() {
					cluster, err := f.EKS.DescribeClusterWithContext(
						ctx,
						&eks.DescribeClusterInput{
							Name: aws.String(*eksClusterName),
						},
					)
					if err != nil {
						// TODO: handle error here
					}
					mu.Lock()
					clusters = append(clusters, cluster.Cluster)
					mu.Unlock()
				}()
			}
			wg.Done()
			return true
		},
	)

	if err != nil {
		return nil, trace.Wrap(err)
	}
	return filterByLabels(f.Labels, clusters)
}

func eksTagsToLabels(tags map[string]*string) map[string]string {
	labels := make(map[string]string)
	for key, valuePtr := range tags {
		if types.IsValidLabelKey(key) {
			labels[key] = aws.StringValue(valuePtr)
		} else {
			log.Debugf("Skipping EC2 tag %q, not a valid label key", key)
		}
	}
	return labels
}

func filterByLabels(labels types.Labels, clusters []*eks.Cluster) ([]*eks.Cluster, error) {
	var result []*eks.Cluster
	for _, cluster := range clusters {
		instanceLabels := eksTagsToLabels(cluster.Tags)
		match, _, err := services.MatchLabels(labels, instanceLabels)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		if !match {
			continue
		}
		result = append(result, cluster)
	}
	return result, nil
}
