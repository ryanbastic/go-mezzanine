# ErrorModel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Detail** | Pointer to **string** | A human-readable explanation specific to this occurrence of the problem. | [optional] 
**Errors** | Pointer to [**[]ErrorDetail**](ErrorDetail.md) | Optional list of individual error details | [optional] 
**Instance** | Pointer to **string** | A URI reference that identifies the specific occurrence of the problem. | [optional] 
**Status** | Pointer to **int64** | HTTP status code | [optional] 
**Title** | Pointer to **string** | A short, human-readable summary of the problem type. This value should not change between occurrences of the error. | [optional] 
**Type** | Pointer to **string** | A URI reference to human-readable documentation for the error. | [optional] [default to "about:blank"]

## Methods

### NewErrorModel

`func NewErrorModel() *ErrorModel`

NewErrorModel instantiates a new ErrorModel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewErrorModelWithDefaults

`func NewErrorModelWithDefaults() *ErrorModel`

NewErrorModelWithDefaults instantiates a new ErrorModel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ErrorModel) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ErrorModel) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ErrorModel) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ErrorModel) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDetail

`func (o *ErrorModel) GetDetail() string`

GetDetail returns the Detail field if non-nil, zero value otherwise.

### GetDetailOk

`func (o *ErrorModel) GetDetailOk() (*string, bool)`

GetDetailOk returns a tuple with the Detail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetail

`func (o *ErrorModel) SetDetail(v string)`

SetDetail sets Detail field to given value.

### HasDetail

`func (o *ErrorModel) HasDetail() bool`

HasDetail returns a boolean if a field has been set.

### GetErrors

`func (o *ErrorModel) GetErrors() []ErrorDetail`

GetErrors returns the Errors field if non-nil, zero value otherwise.

### GetErrorsOk

`func (o *ErrorModel) GetErrorsOk() (*[]ErrorDetail, bool)`

GetErrorsOk returns a tuple with the Errors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetErrors

`func (o *ErrorModel) SetErrors(v []ErrorDetail)`

SetErrors sets Errors field to given value.

### HasErrors

`func (o *ErrorModel) HasErrors() bool`

HasErrors returns a boolean if a field has been set.

### SetErrorsNil

`func (o *ErrorModel) SetErrorsNil(b bool)`

 SetErrorsNil sets the value for Errors to be an explicit nil

### UnsetErrors
`func (o *ErrorModel) UnsetErrors()`

UnsetErrors ensures that no value is present for Errors, not even an explicit nil
### GetInstance

`func (o *ErrorModel) GetInstance() string`

GetInstance returns the Instance field if non-nil, zero value otherwise.

### GetInstanceOk

`func (o *ErrorModel) GetInstanceOk() (*string, bool)`

GetInstanceOk returns a tuple with the Instance field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstance

`func (o *ErrorModel) SetInstance(v string)`

SetInstance sets Instance field to given value.

### HasInstance

`func (o *ErrorModel) HasInstance() bool`

HasInstance returns a boolean if a field has been set.

### GetStatus

`func (o *ErrorModel) GetStatus() int64`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *ErrorModel) GetStatusOk() (*int64, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *ErrorModel) SetStatus(v int64)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *ErrorModel) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### GetTitle

`func (o *ErrorModel) GetTitle() string`

GetTitle returns the Title field if non-nil, zero value otherwise.

### GetTitleOk

`func (o *ErrorModel) GetTitleOk() (*string, bool)`

GetTitleOk returns a tuple with the Title field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTitle

`func (o *ErrorModel) SetTitle(v string)`

SetTitle sets Title field to given value.

### HasTitle

`func (o *ErrorModel) HasTitle() bool`

HasTitle returns a boolean if a field has been set.

### GetType

`func (o *ErrorModel) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *ErrorModel) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *ErrorModel) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *ErrorModel) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


