package api

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func Success(data any) Response {
	return Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
}

func Failure(code int, msg string) Response {
	return Response{
		Code: code,
		Msg:  msg,
		Data: map[string]any{},
	}
}
