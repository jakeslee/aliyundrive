package http

type Method string

const (
	Get     Method = "GET"
	Post    Method = "POST"
	Put     Method = "PUT"
	Delete  Method = "DELETE"
	Patch   Method = "PATCH"
	Head    Method = "HEAD"
	Options Method = "OPTIONS"
)

type Request interface {
	GetHttpMethod() Method
	GetUrl() string
	GetQueryParams() map[string]string
	GetHeaders() map[string]string

	SetHttpMethod(method Method) Request
	SetUrl(url string) Request
}

type BaseRequest struct {
	httpMethod Method
	url        string
	headers    map[string]string
	params     map[string]string
	endpoint   string
}

func (receiver *BaseRequest) Init(endpoint string) *BaseRequest {
	receiver.headers = make(map[string]string)
	receiver.params = make(map[string]string)

	receiver.endpoint = endpoint

	return receiver
}

func (receiver *BaseRequest) GetHttpMethod() Method {
	return receiver.httpMethod
}

func (receiver *BaseRequest) GetUrl() string {
	return receiver.endpoint + receiver.url
}

func (receiver *BaseRequest) GetQueryParams() map[string]string {
	return receiver.params
}

func (receiver *BaseRequest) GetHeaders() map[string]string {
	return receiver.headers
}

func (receiver *BaseRequest) SetUrl(url string) Request {
	receiver.url = url

	return receiver
}

func (receiver *BaseRequest) SetHttpMethod(method Method) Request {
	receiver.httpMethod = method

	return receiver
}
