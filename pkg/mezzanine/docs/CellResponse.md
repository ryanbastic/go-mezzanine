# CellResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AddedId** | **int64** | Auto-incremented ID | 
**Body** | **interface{}** |  | 
**ColumnName** | **string** | Column name | 
**CreatedAt** | **time.Time** | Creation timestamp | 
**RefKey** | **int64** | Reference key version | 
**RowKey** | **string** | Row key UUID | 

## Methods

### NewCellResponse

`func NewCellResponse(addedId int64, body interface{}, columnName string, createdAt time.Time, refKey int64, rowKey string, ) *CellResponse`

NewCellResponse instantiates a new CellResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCellResponseWithDefaults

`func NewCellResponseWithDefaults() *CellResponse`

NewCellResponseWithDefaults instantiates a new CellResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CellResponse) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CellResponse) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CellResponse) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CellResponse) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAddedId

`func (o *CellResponse) GetAddedId() int64`

GetAddedId returns the AddedId field if non-nil, zero value otherwise.

### GetAddedIdOk

`func (o *CellResponse) GetAddedIdOk() (*int64, bool)`

GetAddedIdOk returns a tuple with the AddedId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddedId

`func (o *CellResponse) SetAddedId(v int64)`

SetAddedId sets AddedId field to given value.


### GetBody

`func (o *CellResponse) GetBody() interface{}`

GetBody returns the Body field if non-nil, zero value otherwise.

### GetBodyOk

`func (o *CellResponse) GetBodyOk() (*interface{}, bool)`

GetBodyOk returns a tuple with the Body field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBody

`func (o *CellResponse) SetBody(v interface{})`

SetBody sets Body field to given value.


### SetBodyNil

`func (o *CellResponse) SetBodyNil(b bool)`

 SetBodyNil sets the value for Body to be an explicit nil

### UnsetBody
`func (o *CellResponse) UnsetBody()`

UnsetBody ensures that no value is present for Body, not even an explicit nil
### GetColumnName

`func (o *CellResponse) GetColumnName() string`

GetColumnName returns the ColumnName field if non-nil, zero value otherwise.

### GetColumnNameOk

`func (o *CellResponse) GetColumnNameOk() (*string, bool)`

GetColumnNameOk returns a tuple with the ColumnName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetColumnName

`func (o *CellResponse) SetColumnName(v string)`

SetColumnName sets ColumnName field to given value.


### GetCreatedAt

`func (o *CellResponse) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *CellResponse) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *CellResponse) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetRefKey

`func (o *CellResponse) GetRefKey() int64`

GetRefKey returns the RefKey field if non-nil, zero value otherwise.

### GetRefKeyOk

`func (o *CellResponse) GetRefKeyOk() (*int64, bool)`

GetRefKeyOk returns a tuple with the RefKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefKey

`func (o *CellResponse) SetRefKey(v int64)`

SetRefKey sets RefKey field to given value.


### GetRowKey

`func (o *CellResponse) GetRowKey() string`

GetRowKey returns the RowKey field if non-nil, zero value otherwise.

### GetRowKeyOk

`func (o *CellResponse) GetRowKeyOk() (*string, bool)`

GetRowKeyOk returns a tuple with the RowKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRowKey

`func (o *CellResponse) SetRowKey(v string)`

SetRowKey sets RowKey field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


