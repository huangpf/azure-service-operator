// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package storageaccount

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azurev1alpha1 "github.com/Azure/azure-service-operator/api/v1alpha1"
	"github.com/Azure/azure-service-operator/pkg/resourcemanager"
	"github.com/Azure/go-autorest/autorest"
)

// New returns an instance of the Storage Account Client
func New() *azureStorageManager {
	return &azureStorageManager{}
}

type StorageManager interface {
	CreateStorage(ctx context.Context, groupName string,
		storageAccountName string,
		location string,
		sku azurev1alpha1.StorageSku,
		kind azurev1alpha1.StorageKind,
		tags map[string]*string,
		accessTier azurev1alpha1.StorageAccessTier,
		enableHTTPsTrafficOnly *bool, dataLakeEnabled *bool) (result storage.Account, err error)

	// Get gets the description of the specified storage account.
	// Parameters:
	// resourceGroupName - name of the resource group within the azure subscription.
	// accountName - the name of the storage account
	GetStorage(ctx context.Context, resourceGroupName string, accountName string) (result storage.Account, err error)

	// DeleteStorage removes the storage account
	// Parameters:
	// resourceGroupName - name of the resource group within the azure subscription.
	// accountName - the name of the storage account
	DeleteStorage(ctx context.Context, groupName string, storageAccountName string) (result autorest.Response, err error)

	ListKeys(ctx context.Context, groupName string, storageAccountName string) (result storage.AccountListKeysResult, err error)

	resourcemanager.ARMClient
}
