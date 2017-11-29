package main

type RegistrationResponse struct {
	Ctrl CtrlPayload `json:"ctrl"`
}

type CtrlPayload struct {
	Code   int           `json:"code"`
	Text   string        `json:"text"`
	Params ParamsPayload `json:"params"`
}

type ParamsPayload struct {
	Token string `json:"token"`
	User  string `json:"user"`
}
