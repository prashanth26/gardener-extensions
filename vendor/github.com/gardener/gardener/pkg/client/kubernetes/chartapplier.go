// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package kubernetes

import (
	"context"

	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ChartApplier is an interface that describes needed methods that render and apply
// Helm charts in Kubernetes clusters.
type ChartApplier interface {
	chartrenderer.ChartRenderer
	ApplierInterface

	ApplyChartWithOptions(context.Context, string, string, string, map[string]interface{}, map[string]interface{}, ApplierOptions) error
	ApplyChart(context.Context, string, string, string, map[string]interface{}, map[string]interface{}) error
	ApplyChartInNamespaceWithOptions(context.Context, string, string, string, map[string]interface{}, map[string]interface{}, ApplierOptions) error
	ApplyChartInNamespace(context.Context, string, string, string, map[string]interface{}, map[string]interface{}) error
}

// chartApplier is a structure that contains a chart renderer and a manifest applier.
type chartApplier struct {
	chartrenderer.ChartRenderer
	ApplierInterface
}

// NewChartApplierForConfig returns a new chart applier based on the given REST config.
func NewChartApplierForConfig(config *rest.Config) (ChartApplier, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	renderer, err := chartrenderer.New(clientset)
	if err != nil {
		return nil, err
	}
	applier, err := NewApplierForConfig(config)
	if err != nil {
		return nil, err
	}

	return NewChartApplier(renderer, applier), nil
}

// NewChartApplier returns a new chart applier.
func NewChartApplier(renderer chartrenderer.ChartRenderer, applier ApplierInterface) ChartApplier {
	return &chartApplier{renderer, applier}
}

// ApplyChartWithOptions takes a path to a chart <chartPath>, name of the release <name>,
// release's namespace <namespace> and two maps <defaultValues>, <additionalValues>, and renders the template
// based on the merged result of both value maps. The resulting manifest will be applied to the cluster the
// Kubernetes client has been created for.
// <options> determines how the apply logic is executed.
func (c *chartApplier) ApplyChartWithOptions(ctx context.Context, chartPath, namespace, name string, defaultValues, additionalValues map[string]interface{}, options ApplierOptions) error {
	manifestReader, err := c.manifestReader(chartPath, namespace, name, defaultValues, additionalValues)
	if err != nil {
		return err
	}
	return c.ApplyManifest(ctx, manifestReader, options)
}

// ApplyChartInNamespaceWithOptions is the same as ApplyChart except that it forces the namespace for chart objects when applying the chart, this is because sometimes native chart
// objects do not come with a Release.Namespace option and leave the namespace field empty.
func (c *chartApplier) ApplyChartInNamespaceWithOptions(ctx context.Context, chartPath, namespace, name string, defaultValues, additionalValues map[string]interface{}, options ApplierOptions) error {
	manifestReader, err := c.manifestReader(chartPath, namespace, name, defaultValues, additionalValues)
	if err != nil {
		return err
	}

	nameSpaceSettingsReader := NewNamespaceSettingReader(manifestReader, namespace)
	return c.ApplyManifest(ctx, nameSpaceSettingsReader, DefaultApplierOptions)
}

// ApplyChart takes a path to a chart <chartPath>, name of the release <name>,
// release's namespace <namespace> and two maps <defaultValues>, <additionalValues>, and renders the template
// based on the merged result of both value maps. The resulting manifest will be applied to the cluster the
// Kubernetes client has been created for.
func (c *chartApplier) ApplyChart(ctx context.Context, chartPath, namespace, name string, defaultValues, additionalValues map[string]interface{}) error {
	return c.ApplyChartWithOptions(ctx, chartPath, namespace, name, defaultValues, additionalValues, DefaultApplierOptions)
}

// ApplyChartInNamespace is the same as ApplyChart except that it forces the namespace for chart objects when applying the chart, this is because sometimes native chart
// objects do not come with a Release.Namespace option and leave the namespace field empty.
func (c *chartApplier) ApplyChartInNamespace(ctx context.Context, chartPath, namespace, name string, defaultValues, additionalValues map[string]interface{}) error {
	return c.ApplyChartInNamespaceWithOptions(ctx, chartPath, namespace, name, defaultValues, additionalValues, DefaultApplierOptions)
}

func (c *chartApplier) manifestReader(chartPath, namespace, name string, defaultValues, additionalValues map[string]interface{}) (UnstructuredReader, error) {
	release, err := c.Render(chartPath, name, namespace, utils.MergeMaps(defaultValues, additionalValues))
	if err != nil {
		return nil, err
	}
	return NewManifestReader(release.Manifest()), nil
}