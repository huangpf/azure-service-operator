// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package resourcegroups

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-service-operator/api/v1alpha1"
	azurev1alpha1 "github.com/Azure/azure-service-operator/api/v1alpha1"
	"github.com/Azure/azure-service-operator/pkg/errhelp"
	helpers "github.com/Azure/azure-service-operator/pkg/helpers"
	"github.com/Azure/azure-service-operator/pkg/resourcemanager"
	"k8s.io/apimachinery/pkg/runtime"
)

func (g *AzureResourceGroupManager) Ensure(ctx context.Context, obj runtime.Object, opts ...resourcemanager.ConfigOption) (bool, error) {

	instance, err := g.convert(obj)
	if err != nil {
		return false, err
	}
	resourcegroupLocation := instance.Spec.Location
	resourcegroupName := instance.ObjectMeta.Name
	instance.Status.Provisioning = true

	group, err := g.CreateGroup(ctx, resourcegroupName, resourcegroupLocation)
	if err != nil {
		instance.Status.Provisioned = false
		instance.Status.Message = err.Error()

		// handle special cases that won't work without a change to spec
		if group.StatusCode == http.StatusBadRequest {
			instance.Status.Provisioning = false
			return true, nil
		}

		return false, fmt.Errorf("ResourceGroup create error %v", err)

	}

	instance.Status.Provisioned = true
	instance.Status.Provisioning = false
	instance.Status.Message = resourcemanager.SuccessMsg
	instance.Status.ResourceId = *group.ID

	return true, nil
}

func (g *AzureResourceGroupManager) Delete(ctx context.Context, obj runtime.Object, opts ...resourcemanager.ConfigOption) (bool, error) {
	instance, err := g.convert(obj)
	if err != nil {
		return false, err
	}

	resourcegroup := instance.ObjectMeta.Name

	_, err = g.DeleteGroup(ctx, resourcegroup)
	if err != nil {

		catch := []string{
			errhelp.ResourceGroupNotFoundErrorCode,
			errhelp.AsyncOpIncompleteError,
		}
		err = errhelp.NewAzureError(err)
		if azerr, ok := err.(*errhelp.AzureError); ok {
			if helpers.ContainsString(catch, azerr.Type) {
				return false, nil
			}
		}

		return true, fmt.Errorf("ResourceGroup delete error %v", err)

	}

	return true, nil
}

func (g *AzureResourceGroupManager) GetParents(obj runtime.Object) ([]resourcemanager.KubeParent, error) {
	return nil, nil
}

func (g *AzureResourceGroupManager) GetStatus(obj runtime.Object) (*v1alpha1.ASOStatus, error) {
	instance, err := g.convert(obj)
	if err != nil {
		return nil, err
	}
	return &instance.Status, nil
}

func (g *AzureResourceGroupManager) convert(obj runtime.Object) (*azurev1alpha1.ResourceGroup, error) {
	local, ok := obj.(*azurev1alpha1.ResourceGroup)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
