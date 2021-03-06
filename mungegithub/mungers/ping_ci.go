/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

package mungers

import (
	"time"

	"k8s.io/contrib/mungegithub/github"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// PingCIMunger looks for situations CI (Travis | Shippable) has flaked for some
// reason and we want to re-run them.  Achieves this by closing and re-opening the pr
type PingCIMunger struct{}

func init() {
	RegisterMungerOrDie(PingCIMunger{})
}

// Name is the name usable in --pr-mungers
func (PingCIMunger) Name() string { return "ping-ci" }

// Initialize will initialize the munger
func (PingCIMunger) Initialize(config *github.Config) error { return nil }

// EachLoop is called at the start of every munge loop
func (PingCIMunger) EachLoop() error { return nil }

// AddFlags will add any request flags to the cobra `cmd`
func (PingCIMunger) AddFlags(cmd *cobra.Command, config *github.Config) {}

// Munge is the workhorse the will actually make updates to the PR
func (PingCIMunger) Munge(obj *github.MungeObject) {
	if !obj.IsPR() {
		return
	}

	if !obj.HasLabel("lgtm") {
		return
	}
	mergeable, err := obj.IsMergeable()
	if err != nil {
		glog.V(2).Infof("Skipping %d - problem determining mergeability", *obj.Issue.Number)
		return
	}
	if !mergeable {
		glog.V(2).Infof("Skipping %d - not mergeable", *obj.Issue.Number)
		return
	}
	if status, err := obj.GetStatus([]string{jenkinsCIContext, travisContext}); err == nil && status != "incomplete" {
		glog.V(2).Info("Have %s status - skipping ping CI", jenkinsCIContext)
		return
	}
	status, err := obj.GetStatus([]string{shippableContext, travisContext})
	if err != nil {
		glog.Errorf("unexpected error getting status: %v", err)
		return
	}
	if status == "incomplete" {
		msg := "Continuous integration appears to have missed, closing and re-opening to trigger it"
		obj.WriteComment(msg)

		obj.ClosePR()
		time.Sleep(5 * time.Second)
		obj.OpenPR(10)
	}
}
