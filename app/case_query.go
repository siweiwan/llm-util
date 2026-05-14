package app

type CaseQuery struct {
	AppId  string `json:"id"`
	AppKey string `json:"key"`
}

func (c *CaseQuery) Id() string {
	return c.AppId
}

func (c *CaseQuery) Key() string {
	return c.AppKey
}

func (c *CaseQuery) SendRequest(request interface{}) (interface{}, error) {
	return nil, nil
}
