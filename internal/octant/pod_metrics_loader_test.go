/*
 * Copyright (c) 2019 VMware, Inc. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package octant_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/vmware-tanzu/octant/internal/cluster"
	clusterFake "github.com/vmware-tanzu/octant/internal/cluster/fake"
	"github.com/vmware-tanzu/octant/internal/gvk"
	"github.com/vmware-tanzu/octant/internal/octant"
	octantFake "github.com/vmware-tanzu/octant/internal/octant/fake"
	"github.com/vmware-tanzu/octant/internal/testutil"
)

func TestNewClusterPodMetricsLoader(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	clusterClient := clusterFake.NewMockClientInterface(controller)

	tests := []struct {
		name          string
		clusterClient cluster.ClientInterface
		wantErr       bool
	}{
		{
			name:          "with a cluster client",
			clusterClient: clusterClient,
		},
		{
			name:          "without a cluster client",
			clusterClient: nil,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := octant.NewClusterPodMetricsLoader(tt.clusterClient)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClusterPodMetricsLoader_Load(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	clusterClient := clusterFake.NewMockClientInterface(controller)

	m := testutil.ToUnstructured(t, testutil.CreatePodMetrics("pod"))

	tests := []struct {
		name          string
		clusterClient cluster.ClientInterface
		options       []octant.ClusterPodMetricsLoaderOption
		want          *unstructured.Unstructured
		wantErr       bool
	}{
		{
			name: "in general",
			options: []octant.ClusterPodMetricsLoaderOption{
				func(loader *octant.ClusterPodMetricsLoader) {
					crud := octantFake.NewMockPodMetricsCRUD(controller)
					crud.EXPECT().
						Get("test", "pod").
						Return(m, nil)
					loader.PodMetricsCRUD = crud
				},
			},
			clusterClient: clusterClient,
			want:          m,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pml, err := octant.NewClusterPodMetricsLoader(tt.clusterClient, tt.options...)
			require.NoError(t, err)

			got, err := pml.Load("test", "pod")

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClusterPodMetricsLoader_SupportsMetrics(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	tests := []struct {
		name          string
		clusterClient cluster.ClientInterface
		want          bool
		wantErr       bool
	}{
		{
			name:          "cluster supports pod metrics",
			clusterClient: initClusterClientWithPodMetrics(controller),
			want:          true,
		},
		{
			name:          "cluster does not support pod metrics",
			clusterClient: initClusterClientWithoutPodMetrics(controller),
			want:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pml, err := octant.NewClusterPodMetricsLoader(tt.clusterClient)
			require.NoError(t, err)

			got, err := pml.SupportsMetrics()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func initClusterClientWithPodMetrics(controller *gomock.Controller) *clusterFake.MockClientInterface {
	apiResourceLists := []*metav1.APIResourceList{
		{
			GroupVersion: gvk.PodMetrics.GroupVersion().String(),
			APIResources: []metav1.APIResource{
				{
					Kind: gvk.PodMetrics.Kind,
				},
			},
		},
	}

	discoveryClient := clusterFake.NewMockDiscoveryInterface(controller)
	discoveryClient.EXPECT().ServerPreferredNamespacedResources().Return(apiResourceLists, nil)

	clusterClient := clusterFake.NewMockClientInterface(controller)
	clusterClient.EXPECT().DiscoveryClient().Return(discoveryClient, nil)

	return clusterClient
}

func initClusterClientWithoutPodMetrics(controller *gomock.Controller) *clusterFake.MockClientInterface {
	var apiResourceLists []*metav1.APIResourceList

	discoveryClient := clusterFake.NewMockDiscoveryInterface(controller)
	discoveryClient.EXPECT().ServerPreferredNamespacedResources().Return(apiResourceLists, nil)

	clusterClient := clusterFake.NewMockClientInterface(controller)
	clusterClient.EXPECT().DiscoveryClient().Return(discoveryClient, nil)

	return clusterClient
}
