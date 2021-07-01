package adb

import (
	goadb "github.com/zach-klippenstein/goadb"
)

type Client struct {
	adb *goadb.Adb
	dev *goadb.Device
}

func CreateClient() (*Client, error) {
	adb, err := goadb.NewWithConfig(goadb.ServerConfig{})
	if err != nil {
		return nil, err
	}

	return &Client{
		adb: adb,
	}, nil
}

func (client *Client) Close() {

}

func (client *Client) SelectAnyUsbDevice() {
	client.dev = client.adb.Device(
		goadb.AnyUsbDevice(),
	)
}

func (client *Client) Device() *goadb.Device {
	return client.dev
}
