// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package azuresqldb

import (
	"context"
	"fmt"

	azurev1alpha1 "github.com/Azure/azure-service-operator/api/v1alpha1"
	"github.com/Azure/azure-service-operator/pkg/errhelp"
	"github.com/Azure/azure-service-operator/pkg/helpers"
	"github.com/Azure/azure-service-operator/pkg/resourcemanager"
	azuresqlshared "github.com/Azure/azure-service-operator/pkg/resourcemanager/azuresql/azuresqlshared"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Ensure creates an AzureSqlDb
func (db *AzureSqlDbManager) Ensure(ctx context.Context, obj runtime.Object, opts ...resourcemanager.ConfigOption) (bool, error) {

	instance, err := db.convert(obj)
	if err != nil {
		return false, err
	}

	// set a spec hash if one hasn't been set
	hash := helpers.Hash256(instance.Spec)
	if instance.Status.SpecHash == hash && instance.Status.Provisioned {
		instance.Status.RequestedAt = nil
		return true, nil
	}

	if instance.Status.SpecHash == "" {
		instance.Status.SpecHash = hash
	}

	location := instance.Spec.Location
	groupName := instance.Spec.ResourceGroup
	server := instance.Spec.Server
	dbName := instance.Name
	dbEdition := instance.Spec.Edition

	// convert kube labels to expected tag format
	labels := helpers.LabelsToTags(instance.GetLabels())

	azureSQLDatabaseProperties := azuresqlshared.SQLDatabaseProperties{
		DatabaseName: dbName,
		Edition:      dbEdition,
	}

	instance.Status.Provisioning = true
	instance.Status.Provisioned = false

	dbGet, err := db.GetDB(ctx, groupName, server, dbName)
	if err == nil {

		// db exists, we have successfully provisioned everything
		instance.Status.Provisioning = false
		instance.Status.Provisioned = true
		instance.Status.State = string(*dbGet.Status)
		instance.Status.Message = resourcemanager.SuccessMsg
		instance.Status.ResourceId = *dbGet.ID
		return true, nil
	} else {
		azerr := errhelp.NewAzureErrorAzureError(err)
		ignore := []string{
			errhelp.NotFoundErrorCode,
			errhelp.ResourceNotFound,
			errhelp.ResourceGroupNotFoundErrorCode,
		}
		if !helpers.ContainsString(ignore, azerr.Type) {
			instance.Status.Message = err.Error()
			instance.Status.Provisioning = false
			return false, fmt.Errorf("AzureSqlDb GetDB error %v", err)
		}
	}

	resp, err := db.CreateOrUpdateDB(ctx, groupName, location, server, labels, azureSQLDatabaseProperties)
	if err != nil {
		instance.Status.Message = err.Error()
		azerr := errhelp.NewAzureErrorAzureError(err)

		// resource request has been sent to ARM
		if azerr.Type == errhelp.AsyncOpIncompleteError {
			instance.Status.Provisioning = true
			return false, nil
		}

		// the errors that can arise during reconcilliation where we simply requeue
		catch := []string{
			errhelp.ResourceGroupNotFoundErrorCode,
			errhelp.ParentNotFoundErrorCode,
		}
		if helpers.ContainsString(catch, azerr.Type) {
			instance.Status.Provisioning = false
			return false, nil
		}

		// assertion that a 404 error implies that the Azure SQL server hasn't been provisioned yet
		if resp != nil && resp.StatusCode == 404 {
			instance.Status.Message = fmt.Sprintf("Waiting for SQL Server %s to provision", server)
			instance.Status.Provisioning = false
			return false, nil
		}

		return true, fmt.Errorf("AzureSqlDb CreateOrUpdate error %v", err)
	}

	return false, nil
}

// Delete drops a AzureSqlDb
func (db *AzureSqlDbManager) Delete(ctx context.Context, obj runtime.Object, opts ...resourcemanager.ConfigOption) (bool, error) {
	instance, err := db.convert(obj)
	if err != nil {
		return false, err
	}

	groupName := instance.Spec.ResourceGroup
	server := instance.Spec.Server
	dbName := instance.ObjectMeta.Name

	_, err = db.DeleteDB(ctx, groupName, server, dbName)
	if err != nil {
		if errhelp.IsStatusCode204(err) {
			// Database does not exist
			return false, nil
		}

		return true, fmt.Errorf("AzureSqlDb delete error %v", err)
	}

	return false, nil
}

// GetParents returns the parents of AzureSqlDatabase
func (db *AzureSqlDbManager) GetParents(obj runtime.Object) ([]resourcemanager.KubeParent, error) {
	instance, err := db.convert(obj)
	if err != nil {
		return nil, err
	}

	rgKey := types.NamespacedName{Name: instance.Spec.ResourceGroup, Namespace: instance.Namespace}
	key := types.NamespacedName{Name: instance.Spec.Server, Namespace: instance.Namespace}

	return []resourcemanager.KubeParent{
		{Key: key, Target: &azurev1alpha1.AzureSqlServer{}},
		{Key: rgKey, Target: &azurev1alpha1.ResourceGroup{}},
	}, nil
}

func (g *AzureSqlDbManager) GetStatus(obj runtime.Object) (*azurev1alpha1.ASOStatus, error) {
	instance, err := g.convert(obj)
	if err != nil {
		return nil, err
	}
	return &instance.Status, nil
}

func (*AzureSqlDbManager) convert(obj runtime.Object) (*azurev1alpha1.AzureSqlDatabase, error) {
	local, ok := obj.(*azurev1alpha1.AzureSqlDatabase)
	if !ok {
		return nil, fmt.Errorf("failed type assertion on kind: %s", obj.GetObjectKind().GroupVersionKind().String())
	}
	return local, nil
}
