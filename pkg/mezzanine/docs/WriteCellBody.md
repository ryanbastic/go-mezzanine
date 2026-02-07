# WriteCellBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Body** | **interface{}** |  | 
**ColumnName** | **string** | Column name | 
**RefKey** | **int64** | Reference key version | 
**RowKey** | **string** | Row key UUID | 

## Methods

### NewWriteCellBody

`func NewWriteCellBody(body interface{}, columnName string, refKey int64, rowKey string, ) *WriteCellBody`

NewWriteCellBody instantiates a new WriteCellBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWriteCellBodyWithDefaults

`func NewWriteCellBodyWithDefaults() *WriteCellBody`

NewWriteCellBodyWithDefaults instantiates a new WriteCellBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *WriteCellBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *WriteCellBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *WriteCellBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *WriteCellBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBody

`func (o *WriteCellBody) GetBody() interface{}`

GetBody returns the Body field if non-nil, zero value otherwise.

### GetBodyOk

`func (o *WriteCellBody) GetBodyOk() (*interface{}, bool)`

GetBodyOk returns a tuple with the Body field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBody

`func (o *WriteCellBody) SetBody(v interface{})`

SetBody sets Body field to given value.


### SetBodyNil

`func (o *WriteCellBody) SetBodyNil(b bool)`

 SetBodyNil sets the value for Body to be an explicit nil

### UnsetBody
`func (o *WriteCellBody) UnsetBody()`

UnsetBody ensures that no value is present for Body, not even an explicit nil
### GetColumnName

`func (o *WriteCellBody) GetColumnName() string`

GetColumnName returns the ColumnName field if non-nil, zero value otherwise.

### GetColumnNameOk

`func (o *WriteCellBody) GetColumnNameOk() (*string, bool)`

GetColumnNameOk returns a tuple with the ColumnName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetColumnName

`func (o *WriteCellBody) SetColumnName(v string)`

SetColumnName sets ColumnName field to given value.


### GetRefKey

`func (o *WriteCellBody) GetRefKey() int64`

GetRefKey returns the RefKey field if non-nil, zero value otherwise.

### GetRefKeyOk

`func (o *WriteCellBody) GetRefKeyOk() (*int64, bool)`

GetRefKeyOk returns a tuple with the RefKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefKey

`func (o *WriteCellBody) SetRefKey(v int64)`

SetRefKey sets RefKey field to given value.


### GetRowKey

`func (o *WriteCellBody) GetRowKey() string`

GetRowKey returns the RowKey field if non-nil, zero value otherwise.

### GetRowKeyOk

`func (o *WriteCellBody) GetRowKeyOk() (*string, bool)`

GetRowKeyOk returns a tuple with the RowKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRowKey

`func (o *WriteCellBody) SetRowKey(v string)`

SetRowKey sets RowKey field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


