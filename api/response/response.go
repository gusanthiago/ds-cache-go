package response

type WebResponse struct {
	Code   int         `json:"code"`
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
}

type CacheResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
