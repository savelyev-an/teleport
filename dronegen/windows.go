// Copyright 2021 Gravitational, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "path"

func newWindowsPipeline(name string) pipeline {
	p := newExecPipeline(name)
	p.Workspace.Path = path.Join("C:/Drone/Workspace", name)
	p.Concurrency.Limit = 1
	p.Platform = platform{OS: "windows", Arch: "amd64"}
	return p
}

func windowsPushPipeline() pipeline {
	p := newWindowsPipeline("push-build-native-windows-amd64")
	p.Trigger = trigger{
		Event:  triggerRef{Include: []string{"push"}, Exclude: []string{"pull_request"}},
		Branch: triggerRef{Include: []string{"master", "branch/*", "tcsc/build-windows*"}},
		Repo:   triggerRef{Include: []string{"gravitational/*"}},
	}
	p.Steps = []step{
		{
			Name: "Check out code",
			Environment: map[string]value{
				"WORKSPACE_DIR":      {raw: p.Workspace.Path},
				"GITHUB_PRIVATE_KEY": {fromSecret: "GITHUB_PRIVATE_KEY"},
			},
			Commands: []string{
				`$ErrorActionPreference = 'Stop'`,
				`$TeleportSrc = "$Env:WORKSPACE_DIR/go/src/github.com/gravitational/teleport"`,
				`New-Item -Path "$TeleportSrc" -ItemType Directory | Out-Null`,
				`cd $TeleportSrc`,
				`git clone https://github.com/gravitational/${DRONE_REPO_NAME}.git .`,
				`git checkout ${DRONE_TAG:-$DRONE_COMMIT}`,
			},
		},
	}

	return p
}
