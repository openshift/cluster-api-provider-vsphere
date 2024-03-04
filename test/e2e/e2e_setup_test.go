/*
Copyright 2024 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
	. "sigs.k8s.io/cluster-api/test/framework/ginkgoextensions"
	"sigs.k8s.io/yaml"

	vsphereip "sigs.k8s.io/cluster-api-provider-vsphere/test/framework/ip"
	vspherevcsim "sigs.k8s.io/cluster-api-provider-vsphere/test/framework/vcsim"
	vcsimv1 "sigs.k8s.io/cluster-api-provider-vsphere/test/infrastructure/vcsim/api/v1alpha1"
)

type setupOptions struct {
	additionalIPVariableNames []string
	gatewayIPVariableName     string
}

// SetupOption is a configuration option supplied to Setup.
type SetupOption func(*setupOptions)

// WithIP instructs Setup to allocate another IP and store it into the provided variableName
// NOTE: Setup always allocate an IP for CONTROL_PLANE_ENDPOINT_IP.
func WithIP(variableName string) SetupOption {
	return func(o *setupOptions) {
		o.additionalIPVariableNames = append(o.additionalIPVariableNames, variableName)
	}
}

// WithGateway instructs Setup to store the Gateway IP from IPAM into the provided variableName.
func WithGateway(variableName string) SetupOption {
	return func(o *setupOptions) {
		o.gatewayIPVariableName = variableName
	}
}

// Setup for the specific test.
func Setup(specName string, f func(testSpecificClusterctlConfigPathGetter func() string), opts ...SetupOption) {
	options := &setupOptions{}
	for _, o := range opts {
		o(options)
	}

	var (
		testSpecificClusterctlConfigPath string
		testSpecificIPAddressClaims      vsphereip.AddressClaims
		testSpecificVariables            map[string]string
	)
	BeforeEach(func() {
		Byf("Setting up test env for %s", specName)
		switch testTarget {
		case VCenterTestTarget:
			Byf("Getting IP for %s", strings.Join(append([]string{"CONTROL_PLANE_ENDPOINT_IP"}, options.additionalIPVariableNames...), ","))
			// get IPs from the in cluster address manager
			testSpecificIPAddressClaims, testSpecificVariables = inClusterAddressManager.ClaimIPs(ctx, vsphereip.WithGateway(options.gatewayIPVariableName), vsphereip.WithIP(options.additionalIPVariableNames...))
		case VCSimTestTarget:
			Byf("Getting IP for %s", strings.Join(append([]string{vsphereip.ControlPlaneEndpointIPVariable}, options.additionalIPVariableNames...), ","))

			// get IPs from the vcsim controller
			testSpecificIPAddressClaims, testSpecificVariables = vcsimAddressManager.ClaimIPs(ctx, vsphereip.WithIP(options.additionalIPVariableNames...))

			Byf("Creating a vcsim server for %s", specName)

			// variables for govmomi mode derived from the vCenterSimulator
			vCenterSimulator, err := vspherevcsim.Get(ctx, bootstrapClusterProxy.GetClient())
			Expect(err).ToNot(HaveOccurred(), "Failed to create VCenterSimulator")

			for k, v := range vCenterSimulator.GovmomiVariables() {
				// unset corresponding env variable (that in CI contains VMC data), so we are sure we use the value for vcsim
				if strings.HasPrefix(k, "VSPHERE_") {
					Expect(os.Unsetenv(k)).To(Succeed())
				}

				testSpecificVariables[k] = v
			}

			// variables for govmomi mode derived from envVar.Spec.Cluster
			// NOTE: picking Datacenter, Cluster, Datastore that exists by default in vcsim
			clusterEnvVarSpec := vcsimv1.ClusterEnvVarSpec{
				Datacenter: ptr.To[int32](0), // DC0
				Cluster:    ptr.To[int32](0), // C0
				Datastore:  ptr.To[int32](0), // LocalDS_0
			}

			for k, v := range clusterEnvVarSpec.GovmomiVariables() {
				testSpecificVariables[k] = v
			}
		}

		// Create a new clusterctl config file based on the passed file and add the new variables for the IPs.
		testSpecificClusterctlConfigPath = fmt.Sprintf("%s-%s.yaml", strings.TrimSuffix(clusterctlConfigPath, ".yaml"), specName)
		Byf("Writing a new clusterctl config to %s", testSpecificClusterctlConfigPath)
		copyAndAmendClusterctlConfig(ctx, copyAndAmendClusterctlConfigInput{
			ClusterctlConfigPath: clusterctlConfigPath,
			OutputPath:           testSpecificClusterctlConfigPath,
			Variables:            testSpecificVariables,
		})
	})
	defer AfterEach(func() {
		Byf("Cleaning up test env for %s", specName)
		switch testTarget {
		case VCenterTestTarget:
			// cleanup IPs/controlPlaneEndpoint created by the in cluster ipam provider.
			Expect(inClusterAddressManager.Cleanup(ctx, testSpecificIPAddressClaims)).To(Succeed())
		case VCSimTestTarget:
			// cleanup IPs/controlPlaneEndpoint created by the vcsim controller manager.
			Expect(vcsimAddressManager.Cleanup(ctx, testSpecificIPAddressClaims)).To(Succeed())
		}
	})

	// NOTE: it is required to use a function to pass the testSpecificClusterctlConfigPath value into the test func,
	// so when the test is executed the func could get the value set into the BeforeEach block above.
	// If instead we pass the value directly, the test func will get the value at the moment of the initial parsing of
	// the Ginkgo node tree, which is an empty string (the BeforeEach block above are not run during initial parsing).
	f(func() string { return testSpecificClusterctlConfigPath })
}

// Note: Copy-paste from CAPI below.

// copyAndAmendClusterctlConfigInput is the input for copyAndAmendClusterctlConfig.
type copyAndAmendClusterctlConfigInput struct {
	ClusterctlConfigPath string
	OutputPath           string
	Variables            map[string]string
}

// copyAndAmendClusterctlConfig copies the clusterctl-config from ClusterctlConfigPath to
// OutputPath and adds the given Variables.
func copyAndAmendClusterctlConfig(_ context.Context, input copyAndAmendClusterctlConfigInput) {
	// Read clusterctl config from ClusterctlConfigPath.
	clusterctlConfigFile := &clusterctlConfig{
		Path: input.ClusterctlConfigPath,
	}
	clusterctlConfigFile.read()

	// Overwrite variables.
	if clusterctlConfigFile.Values == nil {
		clusterctlConfigFile.Values = map[string]interface{}{}
	}
	for key, value := range input.Variables {
		clusterctlConfigFile.Values[key] = value
	}

	// Write clusterctl config to OutputPath.
	clusterctlConfigFile.Path = input.OutputPath
	clusterctlConfigFile.write()
}

type clusterctlConfig struct {
	Path   string
	Values map[string]interface{}
}

// write writes a clusterctl config file to disk.
func (c *clusterctlConfig) write() {
	data, err := yaml.Marshal(c.Values)
	Expect(err).ToNot(HaveOccurred(), "Failed to marshal the clusterctl config file")

	Expect(os.WriteFile(c.Path, data, 0600)).To(Succeed(), "Failed to write the clusterctl config file")
}

// read reads a clusterctl config file from disk.
func (c *clusterctlConfig) read() {
	data, err := os.ReadFile(c.Path)
	Expect(err).ToNot(HaveOccurred())

	err = yaml.Unmarshal(data, &c.Values)
	Expect(err).ToNot(HaveOccurred(), "Failed to unmarshal the clusterctl config file")
}
