package providers

type ConfigProvider interface {
	Load(cfg any) error
}
