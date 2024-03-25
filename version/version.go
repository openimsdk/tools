// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"fmt"
	"runtime"

	"github.com/openimsdk/tools/errs"
	"gopkg.in/src-d/go-git.v4"
)

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() Info {
	// These variables typically come from -ldflags settings and in
	// their absence fallback to the settings in ./base.go
	return Info{
		Major:        gitMajor,
		Minor:        gitMinor,
		GitVersion:   gitVersion,
		GitTreeState: gitTreeState,
		GitCommit:    gitCommit,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetClientVersion returns the git version of the OpenIM client repository given a repository URL.
func GetClientVersion(repoURL string) (*OpenIMClientVersion, error) {
	clientVersion, err := getClientVersion(repoURL)
	if err != nil {
		return nil, errs.WrapMsg(err, "failed to get client version", "repoURL", repoURL)
	}
	return &OpenIMClientVersion{
		ClientVersion: clientVersion,
	}, nil
}

func getClientVersion(repoURL string) (string, error) {
	// Temp directory for cloning could be made more unique or cleaned up after use
	tempDir := "/tmp/openim-sdk-core"

	// Consider checking if the repo already exists and just fetch updates instead of cloning every time
	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		return "", errs.WrapMsg(err, "failed to clone OpenIM client repository", "repoURL", repoURL)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", errs.WrapMsg(err, "failed to get head reference", "repoURL", repoURL)
	}

	return ref.Hash().String(), nil
}

// GetSingleVersion returns single version of sealer.
func GetSingleVersion() string {
	return gitVersion
}
