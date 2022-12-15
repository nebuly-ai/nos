package gpu

import "context"

type Client interface {
	GetDevices(ctx context.Context) (DeviceList, Error)
}
