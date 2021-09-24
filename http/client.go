package http

import "github.com/go-resty/resty/v2"

type Client struct {
	client *resty.Client
}

func NewClient() *Client {
	client := resty.New()

	client.SetRetryCount(3)

	return &Client{
		client: client,
	}
}

func (c *Client) Send(request Request, response Response) error {
	r := c.client.R().
		SetHeaders(map[string]string{
			"content-type":    "application/json",
			"user-agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36",
			"origin":          "https://aliyundrive.com",
			"accept":          "*/*",
			"Accept-Language": "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3",
			"Connection":      "keep-alive",
		}).
		SetHeaders(request.GetHeaders()).
		SetQueryParams(request.GetQueryParams()).
		SetBody(request)

	resp, err := r.Execute(string(request.GetHttpMethod()), request.GetUrl())

	if err != nil {
		return err
	}

	return parseFromHTTPResponse(resp, response)
}
