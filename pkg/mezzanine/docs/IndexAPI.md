# \IndexAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**QueryIndex**](IndexAPI.md#QueryIndex) | **Get** /v1/index/{index_name}/{shard_key} | Query secondary index



## QueryIndex

> []IndexEntryResponse QueryIndex(ctx, indexName, shardKey).Execute()

Query secondary index

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ryanbastic/go-mezzanine/pkg/mezzanine"
)

func main() {
	indexName := "indexName_example" // string | Secondary index name
	shardKey := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | Shard key UUID

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.IndexAPI.QueryIndex(context.Background(), indexName, shardKey).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `IndexAPI.QueryIndex``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `QueryIndex`: []IndexEntryResponse
	fmt.Fprintf(os.Stdout, "Response from `IndexAPI.QueryIndex`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**indexName** | **string** | Secondary index name | 
**shardKey** | **string** | Shard key UUID | 

### Other Parameters

Other parameters are passed through a pointer to a apiQueryIndexRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**[]IndexEntryResponse**](IndexEntryResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

