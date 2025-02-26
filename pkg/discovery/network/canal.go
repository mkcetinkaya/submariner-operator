/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

package network

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// nolint:nilnil // Intentional as the purpose is to discover.
func discoverCanalFlannelNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	// TODO: this must be smarter, looking for the canal daemonset, with labels k8s-app=canal
	//  and then the reference on the container volumes:
	//   - configMap:
	//          defaultMode: 420
	//          name: canal-config
	//        name: flannel-cfg
	cm, err := clientSet.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "canal-config", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.WithMessage(err, "error obtaining the \"canal-config\" ConfigMap")
	}

	podCIDR := extractPodCIDRFromNetConfigJSON(cm)

	if podCIDR == nil {
		return nil, nil
	}

	clusterNetwork := &ClusterNetwork{
		NetworkPlugin: constants.NetworkPluginCanalFlannel,
		PodCIDRs:      []string{*podCIDR},
	}

	// Try to detect the service CIDRs using the generic functions
	clusterIPRange, err := findClusterIPRange(clientSet)
	if err != nil {
		return nil, err
	}

	if clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
	}

	return clusterNetwork, nil
}

func extractPodCIDRFromNetConfigJSON(cm *v1.ConfigMap) *string {
	netConfJSON := cm.Data["net-conf.json"]
	if netConfJSON == "" {
		return nil
	}
	var netConf struct {
		Network string `json:"Network"`
		// All the other fields are ignored by Unmarshal
	}

	if err := json.Unmarshal([]byte(netConfJSON), &netConf); err == nil {
		return &netConf.Network
	}

	return nil
}
