package domain

type Method string

const (
	MethodGet     Method = "GET"
	MethodPost    Method = "POST"
	MethodPut     Method = "PUT"
	MethodDelete  Method = "DELETE"
	MethodPatch   Method = "PATCH"
	MethodHead    Method = "HEAD"
	MethodOptions Method = "OPTIONS"
	MethodTrace   Method = "TRACE"
	MethodConnect Method = "CONNECT"
)

func GetValidMethods() []Method {
	return []Method{
		MethodGet,
		MethodPost,
		MethodPut,
		MethodDelete,
		MethodPatch,
		MethodHead,
		MethodOptions,
		MethodTrace,
		MethodConnect,
	}
}

func (m Method) Validate() bool {
	for _, validMethod := range GetValidMethods() {
		if m == validMethod {
			return true
		}
	}
	return false
}
