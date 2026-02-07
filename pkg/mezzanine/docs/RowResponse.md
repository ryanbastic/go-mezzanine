# RowResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Cells** | [**[]CellResponse**](CellResponse.md) | Latest cell per column | 
**RowKey** | **string** | Row key UUID | 

## Methods

### NewRowResponse

`func NewRowResponse(cells []CellResponse, rowKey string, ) *RowResponse`

NewRowResponse instantiates a new RowResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRowResponseWithDefaults

`func NewRowResponseWithDefaults() *RowResponse`

NewRowResponseWithDefaults instantiates a new RowResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RowResponse) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RowResponse) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RowResponse) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RowResponse) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCells

`func (o *RowResponse) GetCells() []CellResponse`

GetCells returns the Cells field if non-nil, zero value otherwise.

### GetCellsOk

`func (o *RowResponse) GetCellsOk() (*[]CellResponse, bool)`

GetCellsOk returns a tuple with the Cells field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCells

`func (o *RowResponse) SetCells(v []CellResponse)`

SetCells sets Cells field to given value.


### SetCellsNil

`func (o *RowResponse) SetCellsNil(b bool)`

 SetCellsNil sets the value for Cells to be an explicit nil

### UnsetCells
`func (o *RowResponse) UnsetCells()`

UnsetCells ensures that no value is present for Cells, not even an explicit nil
### GetRowKey

`func (o *RowResponse) GetRowKey() string`

GetRowKey returns the RowKey field if non-nil, zero value otherwise.

### GetRowKeyOk

`func (o *RowResponse) GetRowKeyOk() (*string, bool)`

GetRowKeyOk returns a tuple with the RowKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRowKey

`func (o *RowResponse) SetRowKey(v string)`

SetRowKey sets RowKey field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


