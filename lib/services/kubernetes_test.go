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

package services

import (
	"fmt"
	"testing"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/utils"

	"github.com/stretchr/testify/require"
)

// TestKubernetesServerUnmarshal verifies an kubernetes server resource can be unmarshaled.
func TestKubernetesServerUnmarshal(t *testing.T) {
	expected, err := types.NewKubernetesServerV3(types.Metadata{
		Name:        "test-kube",
		Description: "Test description",
		Labels:      map[string]string{"env": "dev"},
	}, types.KubernetesServerSpecV3{
		Cluster: &types.KubernetesClusterV3{
			Metadata: types.Metadata{
				Name:        "test-cluster",
				Description: "Test description",
				Labels:      map[string]string{"env": "dev"},
			},
		},
	})
	require.NoError(t, err)
	data, err := utils.ToJSON([]byte(kubeServerYAML))
	require.NoError(t, err)
	actual, err := UnmarshalApp(data)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

// TestKubernetesServernMarshal verifies a marshaled kubernetes resource resource can be unmarshaled back.
func TestKubernetesServerMarshal(t *testing.T) {
	expected, err := types.NewKubernetesServerV3(types.Metadata{
		Name:        "test-kube",
		Description: "Test description",
		Labels:      map[string]string{"env": "dev"},
	}, types.KubernetesServerSpecV3{
		Version: "v3",

		Cluster: &types.KubernetesClusterV3{
			Metadata: types.Metadata{
				Name:        "test-cluster",
				Description: "Test description",
				Labels:      map[string]string{"env": "dev"},
			},
		},
	})
	require.NoError(t, err)
	data, err := MarshalKubeServer(expected)
	fmt.Println(string(data))
	require.NoError(t, err)
	actual, err := UnmarshalApp(data)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

var kubeServerYAML = `kind: app
version: v3
metadata:
  name: test-app
  description: "Test description"
  labels:
    env: dev
spec:
  uri: "http://localhost:8080"`
