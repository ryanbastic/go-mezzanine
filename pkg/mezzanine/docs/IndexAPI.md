# \IndexAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**QueryIndex**](IndexAPI.md#QueryIndex) | **Get** /v1/index/{index_name}/{value} | Query secondary index



## QueryIndex

> []IndexEntryResponse QueryIndex(ctx, indexName, value).Execute()

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
	indexName := "user_by_email" // string | Secondary index name
	value := "alice@example.com" // string | Lookup value (e.g. email address)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.IndexAPI.QueryIndex(context.Background(), indexName, value).Execute()
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
**value** | **string** | Lookup value (e.g. email address) |

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

