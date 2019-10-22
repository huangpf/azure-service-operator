package storages

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/storage/datalake/2019-10-31/storagedatalake"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-service-operator/pkg/resourcemanager/config"
	"github.com/Azure/azure-service-operator/pkg/resourcemanager/iam"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"log"
	"net/http"
)

type azureFileSystemManager struct{}

func (_ *azureFileSystemManager) CreateFileSystem(ctx context.Context, groupName string, filesystemName string, timeout *int32, xMsDate string, datalakeName string) (*autorest.Response, error) {
	client := getFileSystemClient(ctx, groupName, datalakeName)
	// empty parameters are optional request headers (properties, requestid)
	result, err := client.Create(ctx, filesystemName, "", "", timeout, xMsDate)
	if err != nil {
		return nil, err
	}

	return &result, err
}

func (_ *azureFileSystemManager) GetFileSystem(ctx context.Context, groupName string, filesystemName string, timeout *int32, xMsDate string, datalakeName string) (autorest.Response, error) {
	response := autorest.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}
	client := getFileSystemClient(ctx, groupName, datalakeName)

	// empty parameters are optional request headers (continuation, maxresults, requestid)
	list, err := client.List(ctx, filesystemName, "", nil, "", timeout, xMsDate)

	if len(*list.Filesystems) == 0 {
		return response, err
	}

	response = list.Response

	return response, err
}

func (_ *azureFileSystemManager) DeleteFileSystem(ctx context.Context, groupName string, filesystemName string, timeout *int32, xMsDate string, datalakeName string) (autorest.Response, error) {
	client := getFileSystemClient(ctx, groupName, datalakeName)
	// empty parameters are optional request headers (ifmodifiedsince, ifunmodifiedsince, requestid)
	return client.Delete(ctx, filesystemName, "", "", "", timeout, xMsDate)
}

func getFileSystemClient(ctx context.Context, groupName string, accountName string) storagedatalake.FilesystemClient {
	xmsversion := "2019-02-02"
	fsClient := storagedatalake.NewFilesystemClient(xmsversion, accountName)
	adlsClient := getStoragesClient()

	accountKey, err := getAccountKey(ctx, groupName, accountName, adlsClient)
	if err != nil {
		log.Fatalf("failed to get the account key for the authorizer: %v\n", err)
	}

	a, err := iam.GetSharedKeyAuthorizer(accountName, accountKey)

	if err != nil {
		log.Fatalf("failed to initialize authorizer: %v\n", err)
	}
	fsClient.Authorizer = a
	fsClient.AddToUserAgent(config.UserAgent())

	return fsClient
}

func getAccountKey(ctx context.Context, groupName string, accountName string, adlsClient storage.AccountsClient) (accountKey string, err error) {
	keys, err := adlsClient.ListKeys(ctx, groupName, accountName)
	if err != nil {
		return "", err
	}

	for _, key := range *keys.Keys {
		if *key.KeyName == "key1" {
			accountKey = *key.Value
		}
	}
	return accountKey, err
}
