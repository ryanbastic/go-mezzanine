# \ShardsAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetShardCount**](ShardsAPI.md#GetShardCount) | **Get** /v1/shards/count | Get shard count



## GetShardCount

> ShardCountResponse GetShardCount(ctx).Execute()

Get shard count

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

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ShardsAPI.GetShardCount(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ShardsAPI.GetShardCount``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetShardCount`: ShardCountResponse
	fmt.Fprintf(os.Stdout, "Response from `ShardsAPI.GetShardCount`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiGetShardCountRequest struct via the builder pattern


### Return type

[**ShardCountResponse**](ShardCountResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

