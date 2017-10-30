// Copyright 2017 Google Inc.
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

package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

type ExpectedCommand struct {
	Args   []string
	Output string
	Err    error
}

func run(fakeClient *fake.Clientset, expectedCmds []ExpectedCommand, runFn func() error) error {
	getClientset = func(kubeconfigPath, context string) (kubeclient.Interface, error) {
		return fakeClient, nil
	}

	i := 0
	executeCommand = func(args []string) (string, error) {
		if i >= len(expectedCmds) {
			return "", fmt.Errorf("unexpected command: %s", strings.Join(args, " "))
		}
		if !reflect.DeepEqual(args, expectedCmds[i].Args) {
			return "", fmt.Errorf("unexpected command: %s, was expecting   : %s", strings.Join(args, " "), strings.Join(expectedCmds[i].Args, " "))
		}
		output, err := expectedCmds[i].Output, expectedCmds[i].Err
		i++
		return output, err
	}
	err := runFn()
	if err != nil {
		return err
	}
	if i != len(expectedCmds) {
		return fmt.Errorf("expected [commands, outputs, errs] not called: %s", expectedCmds[i:])
	}
	return nil
}

func TestValidateCreateArgs(t *testing.T) {
	// ValidateCreateArgs should return an error with empty options.
	options := CreateOptions{}
	if err := ValidateCreateArgs(&options, []string{}); err == nil {
		t.Errorf("Expected error for emtpy options")
	}

	// ValidateCreateArgs should return an error with missing load balancer name.
	options = CreateOptions{
		IngressFilename: "ingress.yaml",
		GCPProject:      "gcp-project",
	}
	if err := ValidateCreateArgs(&options, []string{}); err == nil {
		t.Errorf("Expected error for missing load balancer name")
	}

	// ValidateCreateArgs should return an error with missing ingress.
	options = CreateOptions{
		GCPProject: "gcp-project",
	}
	if err := ValidateCreateArgs(&options, []string{"lbname"}); err == nil {
		t.Errorf("Expected error for missing ingress")
	}

	// ValidateCreateArgs should return an error with missing gcp project.
	options = CreateOptions{
		IngressFilename: "ingress.yaml",
	}
	if err := ValidateCreateArgs(&options, []string{"lbname"}); err == nil {
		t.Errorf("Expected error for missing gcp project")
	}

	// ValidateCreateArgs should succeed when all arguments are passed as expected.
	options = CreateOptions{
		IngressFilename: "ingress.yaml",
		GCPProject:      "gcp-project",
	}
	if err := ValidateCreateArgs(&options, []string{"lbname"}); err != nil {
		t.Errorf("unexpected error from ValidateCreateArgs: %s", err)
	}
}

func TestCreateIngress(t *testing.T) {
	fakeClient := fake.Clientset{}
	fakeClient.AddReactor("create", "ingresses", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		return true, action.(core.CreateAction).GetObject(), nil
	})
	clusters := []string{"cluster1", "cluster2"}

	runFn := func() ([]string, error) {
		return createIngress("kubeconfig", "../../../testdata/ingress.yaml")
	}
	expectedCommands := []ExpectedCommand{
		{
			Args:   []string{"kubectl", "--kubeconfig=kubeconfig", "config", "get-contexts", "-o=name"},
			Output: strings.Join(clusters, "\n"),
			Err:    nil,
		},
	}
	createClusters, err := run(&fakeClient, expectedCommands, runFn)
	if err != nil {
		t.Errorf("%s", err)
	}
	if !reflect.DeepEqual(createClusters, clusters) {
		t.Errorf("unexpected list of clusters in which ingress was created. expected: %v, got: %v", clusters, createClusters)
	}
	actions := fakeClient.Actions()
	if len(actions) != 2 {
		t.Errorf("Expected 2 actions: Create Ingress 1, Create Ingress 2. Got:%v", actions)
	}
	if !actions[0].Matches("create", "ingresses") {
		t.Errorf("Expected ingress creation.")
	}
	// TODO(G-Harmon): Verify that the ingress matches testdata/ingress.yaml
	if !actions[1].Matches("create", "ingresses") {
		t.Errorf("Expected ingress creation.")
	}
}