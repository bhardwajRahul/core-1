package function

type JSFetchOptionsArg struct {
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	Mode           string            `json:"mode"`
	Credentials    string            `json:"credentials"`
	Cache          string            `json:"cache"`
	Redirect       string            `json:"redirect"`
	Referrer       string            `json:"referrer"`
	ReferrerPolicy string            `json:"referrerPolicy"`
	Integrity      string            `json:"integrity"`
	Keepalive      string            `json:"keepalive"`
	Signal         string            `json:"signal"`
}

type HTTPResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

func NewJSFetcthOptionArg() JSFetchOptionsArg {
	defaultOptions := JSFetchOptionsArg{
		Method:         "GET",
		Headers:        make(map[string]string, 0),
		Body:           "",
		Mode:           "no-cors",
		Credentials:    "omit",
		Cache:          "no-cache",
		Redirect:       "error",
		Referrer:       "",
		ReferrerPolicy: "",
		Integrity:      "",
		Keepalive:      "",
		Signal:         "",
	}
	return defaultOptions
}

type JSSendMailArg struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"htmlBody"`
	TextBody string `json:"textBody"`
}
