# ShardCountResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**NumShards** | **int64** | Number of configured shards | 

## Methods

### NewShardCountResponse

`func NewShardCountResponse(numShards int64, ) *ShardCountResponse`

NewShardCountResponse instantiates a new ShardCountResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewShardCountResponseWithDefaults

`func NewShardCountResponseWithDefaults() *ShardCountResponse`

NewShardCountResponseWithDefaults instantiates a new ShardCountResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ShardCountResponse) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ShardCountResponse) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ShardCountResponse) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ShardCountResponse) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetNumShards

`func (o *ShardCountResponse) GetNumShards() int64`

GetNumShards returns the NumShards field if non-nil, zero value otherwise.

### GetNumShardsOk

`func (o *ShardCountResponse) GetNumShardsOk() (*int64, bool)`

GetNumShardsOk returns a tuple with the NumShards field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNumShards

`func (o *ShardCountResponse) SetNumShards(v int64)`

SetNumShards sets NumShards field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


