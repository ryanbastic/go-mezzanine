# \CellsAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetCell**](CellsAPI.md#GetCell) | **Get** /v1/cells/{row_key}/{column_name}/{ref_key} | Get exact cell version
[**GetCellLatest**](CellsAPI.md#GetCellLatest) | **Get** /v1/cells/{row_key}/{column_name} | Get latest cell version
[**GetRow**](CellsAPI.md#GetRow) | **Get** /v1/cells/{row_key} | Get all latest cells for a row
[**PartitionRead**](CellsAPI.md#PartitionRead) | **Get** /v1/cells/partitionRead | Read a partition of cells
[**WriteCell**](CellsAPI.md#WriteCell) | **Post** /v1/cells | Write a cell



## GetCell

> CellResponse GetCell(ctx, rowKey, columnName, refKey).Execute()

Get exact cell version

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
	rowKey := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | Row key UUID
	columnName := "columnName_example" // string | Column name
	refKey := int64(789) // int64 | Reference key version

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CellsAPI.GetCell(context.Background(), rowKey, columnName, refKey).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CellsAPI.GetCell``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetCell`: CellResponse
	fmt.Fprintf(os.Stdout, "Response from `CellsAPI.GetCell`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**rowKey** | **string** | Row key UUID | 
**columnName** | **string** | Column name | 
**refKey** | **int64** | Reference key version | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetCellRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




### Return type

[**CellResponse**](CellResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetCellLatest

> CellResponse GetCellLatest(ctx, rowKey, columnName).Execute()

Get latest cell version

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
	rowKey := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | Row key UUID
	columnName := "columnName_example" // string | Column name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CellsAPI.GetCellLatest(context.Background(), rowKey, columnName).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CellsAPI.GetCellLatest``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetCellLatest`: CellResponse
	fmt.Fprintf(os.Stdout, "Response from `CellsAPI.GetCellLatest`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**rowKey** | **string** | Row key UUID | 
**columnName** | **string** | Column name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetCellLatestRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**CellResponse**](CellResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetRow

> RowResponse GetRow(ctx, rowKey).Execute()

Get all latest cells for a row

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
	rowKey := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | Row key UUID

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CellsAPI.GetRow(context.Background(), rowKey).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CellsAPI.GetRow``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetRow`: RowResponse
	fmt.Fprintf(os.Stdout, "Response from `CellsAPI.GetRow`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**rowKey** | **string** | Row key UUID | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetRowRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RowResponse**](RowResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PartitionRead

> []CellResponse PartitionRead(ctx).PartitionNumber(partitionNumber).ReadType(readType).CreatedAfter(createdAfter).AddedId(addedId).Limit(limit).Execute()

Read a partition of cells

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
    "time"
	openapiclient "github.com/ryanbastic/go-mezzanine/pkg/mezzanine"
)

func main() {
	partitionNumber := int64(789) // int64 | Partition number
	readType := int64(789) // int64 | Read type
	createdAfter := time.Now() // time.Time | Filter cells created after this timestamp (optional)
	addedId := int64(789) // int64 | Filter cells added after ID (optional)
	limit := int64(789) // int64 | Maximum number of cells to return (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CellsAPI.PartitionRead(context.Background()).PartitionNumber(partitionNumber).ReadType(readType).CreatedAfter(createdAfter).AddedId(addedId).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CellsAPI.PartitionRead``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PartitionRead`: []CellResponse
	fmt.Fprintf(os.Stdout, "Response from `CellsAPI.PartitionRead`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPartitionReadRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **partitionNumber** | **int64** | Partition number | 
 **readType** | **int64** | Read type | 
 **createdAfter** | **time.Time** | Filter cells created after this timestamp | 
 **addedId** | **int64** | Filter cells added after ID | 
 **limit** | **int64** | Maximum number of cells to return | 

### Return type

[**[]CellResponse**](CellResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WriteCell

> CellResponse WriteCell(ctx).WriteCellBody(writeCellBody).Execute()

Write a cell

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
	writeCellBody := *openapiclient.NewWriteCellBody(interface{}(123), "ColumnName_example", int64(123), "RowKey_example") // WriteCellBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.CellsAPI.WriteCell(context.Background()).WriteCellBody(writeCellBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `CellsAPI.WriteCell``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WriteCell`: CellResponse
	fmt.Fprintf(os.Stdout, "Response from `CellsAPI.WriteCell`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiWriteCellRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **writeCellBody** | [**WriteCellBody**](WriteCellBody.md) |  | 

### Return type

[**CellResponse**](CellResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

