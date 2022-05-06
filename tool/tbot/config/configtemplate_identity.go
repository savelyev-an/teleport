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

package config

import (
	"context"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/client/identityfile"
	"github.com/gravitational/teleport/tool/tbot/identity"
	"github.com/gravitational/trace"
)

const defaultIdentityFileName = "identity"

// TemplateIdentity is a config template that generates a Teleport identity
// file that can be used by tsh and tctl.
type TemplateIdentity struct {
	FileName string `yaml:"file_name,omitempty"`
}

func (t *TemplateIdentity) CheckAndSetDefaults() error {
	if t.FileName == "" {
		t.FileName = defaultIdentityFileName
	}

	return nil
}

func (t *TemplateIdentity) Name() string {
	return TemplateIdentityName
}

func (t *TemplateIdentity) Describe() []FileDescription {
	return []FileDescription{
		{
			Name: t.FileName,
		},
	}
}

func (t *TemplateIdentity) Render(ctx context.Context, authClient auth.ClientI, currentIdentity *identity.Identity, destination *DestinationConfig) error {
	dest, err := destination.GetDestination()
	if err != nil {
		return trace.Wrap(err)
	}

	hostCAs, err := authClient.GetCertAuthorities(ctx, types.HostCA, false)
	if err != nil {
		return trace.Wrap(err)
	}

	cfg := identityfile.WriteConfig{
		OutputPath: t.FileName,
		Writer: &BotConfigWriter{
			dest: dest,
		},
		Key:    newClientKey(currentIdentity, hostCAs),
		Format: identityfile.FormatFile,

		// Always overwrite to avoid hitting our no-op Stat() and Remove() functions.
		OverwriteDestination: true,
	}

	files, err := identityfile.Write(cfg)
	if err != nil {
		return trace.Wrap(err)
	}

	log.Debugf("Wrote identity file: %+v", files)

	return nil
}
