package azureblobstorage

import (
	"io"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/facebookgo/rocks-strata/strata"
	"encoding/hex"
	"io/ioutil"
	"strings"
)

type AzureBlobStorage struct {
	client *storage.Client
	blobStorageClient *storage.BlobStorageClient
	containerName string
	prefix string
}

func (azureBlob *AzureBlobStorage) addPrefix(path string) string {
	return azureBlob.prefix + "/" + path
}

func (azureBlob *AzureBlobStorage) removePrefix(path string) string {
	return path[len(azureBlob.prefix)+1:]
}

func NewAzureBlobStorage(accountName string, accountKey string, containerName string, prefix string) (*AzureBlobStorage, error) {
	azureClient, err := storage.NewBasicClient(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	blobClient := azureClient.GetBlobService()

	err = blobClient.CreateContainer(containerName, storage.ContainerAccessTypePrivate)
	if err != nil {
		if !strings.Contains(err.Error(), "The specified container already exists.") {
			return nil, err
		}
	}

	return &AzureBlobStorage{
		client: &azureClient,
		blobStorageClient: &blobClient,
		containerName: containerName,
		prefix: prefix,
	}, nil
}

func (azureBlob *AzureBlobStorage) Get(path string) (io.ReadCloser, error) {
	path = azureBlob.addPrefix(path)

	props, err := azureBlob.blobStorageClient.GetBlobProperties(azureBlob.containerName, path)
	if err != nil {
		return nil, err
	}

	etag := props.Etag

	reader, err := azureBlob.blobStorageClient.GetBlob(azureBlob.containerName, path)

	if reader == nil || err != nil {
		return nil, err
	}

	checksum, err := hex.DecodeString(etag)
	if err != nil {
		return nil, err
	}

	return strata.NewChecksummingReader(reader, checksum), nil
}

func (azureBlob *AzureBlobStorage) Put(path string, data []byte) error {
	path = azureBlob.addPrefix(path)

	err := azureBlob.blobStorageClient.PutBlock(azureBlob.containerName, path, "0", data)

	return err
}

func (azureBlob *AzureBlobStorage) PutReader(path string, reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	return azureBlob.Put(path, data)
}

func (azureBlob *AzureBlobStorage) Delete(path string) error {
	path = azureBlob.addPrefix(path)
	err := azureBlob.blobStorageClient.DeleteBlob(azureBlob.containerName, path, map[string]string{})
	return err
}

func (azureBlob *AzureBlobStorage) List(prefix string, maxSize int) ([]string, error) {
	prefix = azureBlob.addPrefix(prefix)
	pathSeparator := ""
	marker := ""

	items := make([]string, 0, 1000)
	for maxSize > 0 {
		// Don't ask for more than 1000 keys at a time. This makes
		// testing simpler because S3 will return at most 1000 keys even if you
		// ask for more, but s3test will return more than 1000 keys if you ask
		// for more. TODO(agf): Fix this behavior in s3test.
		maxReqSize := 1000
		if maxSize < 1000 {
			maxReqSize = maxSize
		}

		contents, err := azureBlob.blobStorageClient.ListBlobs(azureBlob.containerName,
			storage.ListBlobsParameters{ Prefix: prefix, Delimiter: pathSeparator, Marker: marker, MaxResults: uint(maxReqSize) })

		if err != nil {
			return nil, err
		}
		maxSize -= maxReqSize

		for _, key := range contents.Blobs {
			items = append(items, azureBlob.removePrefix(key.Name))
		}

		if len(items) == maxReqSize {
			marker = contents.NextMarker
		} else {
			break;
		}
	}

	return items, nil

}

// Lock is not implemented
func (azureBlob *AzureBlobStorage) Lock(path string) error {
	return nil
}

// Unlock is not implemented
func (azureBlob *AzureBlobStorage) Unlock(path string) error {
	return nil
}