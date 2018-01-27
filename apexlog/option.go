package apexlog

const (
	DefaultMsgKey = "message"
)

type config struct {
	msgKey string
}

type Option func(c *config)

func WithMsgKey(msgKey string) Option {
	return func(c *config) {
		c.msgKey = msgKey
	}
}
