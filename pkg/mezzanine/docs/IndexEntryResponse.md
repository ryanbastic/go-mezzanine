# IndexEntryResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AddedId** | **int64** | Auto-incremented ID | 
**Body** | **interface{}** |  | 
**CreatedAt** | **time.Time** | Creation timestamp | 
**RowKey** | **string** | Row key UUID | 
**ShardKey** | **string** | Shard key value |

## Methods

### NewIndexEntryResponse

`func NewIndexEntryResponse(addedId int64, body interface{}, createdAt time.Time, rowKey string, shardKey string, ) *IndexEntryResponse`

NewIndexEntryResponse instantiates a new IndexEntryResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIndexEntryResponseWithDefaults

`func NewIndexEntryResponseWithDefaults() *IndexEntryResponse`

NewIndexEntryResponseWithDefaults instantiates a new IndexEntryResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddedId

`func (o *IndexEntryResponse) GetAddedId() int64`

GetAddedId returns the AddedId field if non-nil, zero value otherwise.

### GetAddedIdOk

`func (o *IndexEntryResponse) GetAddedIdOk() (*int64, bool)`

GetAddedIdOk returns a tuple with the AddedId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddedId

`func (o *IndexEntryResponse) SetAddedId(v int64)`

SetAddedId sets AddedId field to given value.


### GetBody

`func (o *IndexEntryResponse) GetBody() interface{}`

GetBody returns the Body field if non-nil, zero value otherwise.

### GetBodyOk

`func (o *IndexEntryResponse) GetBodyOk() (*interface{}, bool)`

GetBodyOk returns a tuple with the Body field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBody

`func (o *IndexEntryResponse) SetBody(v interface{})`

SetBody sets Body field to given value.


### SetBodyNil

`func (o *IndexEntryResponse) SetBodyNil(b bool)`

 SetBodyNil sets the value for Body to be an explicit nil

### UnsetBody
`func (o *IndexEntryResponse) UnsetBody()`

UnsetBody ensures that no value is present for Body, not even an explicit nil
### GetCreatedAt

`func (o *IndexEntryResponse) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *IndexEntryResponse) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *IndexEntryResponse) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetRowKey

`func (o *IndexEntryResponse) GetRowKey() string`

GetRowKey returns the RowKey field if non-nil, zero value otherwise.

### GetRowKeyOk

`func (o *IndexEntryResponse) GetRowKeyOk() (*string, bool)`

GetRowKeyOk returns a tuple with the RowKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRowKey

`func (o *IndexEntryResponse) SetRowKey(v string)`

SetRowKey sets RowKey field to given value.


### GetShardKey

`func (o *IndexEntryResponse) GetShardKey() string`

GetShardKey returns the ShardKey field if non-nil, zero value otherwise.

### GetShardKeyOk

`func (o *IndexEntryResponse) GetShardKeyOk() (*string, bool)`

GetShardKeyOk returns a tuple with the ShardKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetShardKey

`func (o *IndexEntryResponse) SetShardKey(v string)`

SetShardKey sets ShardKey field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


