package function

import (
	"github.com/webhookx-io/webhookx/db/entities"
)

type API struct {
	version string

	Request  *RequestAPI  `json:"request"`
	Response *ResponseAPI `json:"response"`
	Utils    *UtilsAPI    `json:"utils"`
	Log      *LogAPI      `json:"log"`

	ctx *ExecutionContext
	res *ExecutionResult
}

func NewAPI(ctx *ExecutionContext, res *ExecutionResult) *API {
	return &API{
		version:  "0.1.0",
		Request:  NewRequestAPI(ctx, res),
		Utils:    NewUtilsAPI(),
		Log:      NewLogger(),
		Response: NewResponseAPI(ctx, res),
		ctx:      ctx,
		res:      res,
	}
}

func (api *API) GetSource() *entities.Source {
	return api.ctx.Source
}

func (api *API) GetEvent() *entities.Event {
	return api.ctx.Event
}

func (api *API) SetEvent(str string) {
	api.res.Payload = &str
}
