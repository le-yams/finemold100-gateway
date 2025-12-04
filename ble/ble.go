package ble

import (
	"errors"
	"time"

	"golang.org/x/exp/slices"
	"tinygo.org/x/bluetooth"
)

const (
	defaultMTU = 64
)

func Scan(timeout time.Duration) error {
	adapter := bluetooth.DefaultAdapter
	err := adapter.Enable()
	if err != nil {
		return err
	}

	var scannedDevices []string
	start := time.Now()
	println("starting bluetooth scan...")
	return adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		address := result.Address.String()
		if slices.Contains(scannedDevices, address) {
			return
		}
		scannedDevices = append(scannedDevices, address)

		println("device found:", address, "RSSI:", result.RSSI, "Name:", result.LocalName())

		println(" * looking up advertisement manufacturer data:")
		for _, data := range result.AdvertisementPayload.ManufacturerData() {
			println("   - company ID:", data.CompanyID)
			println("   - data:", data.Data)
			println("   - data.string:", string(data.Data))
		}
		result.AdvertisementPayload.ServiceData()

		if timeout > 0 && time.Since(start) > timeout {
			println("stopping scan after timeout:", timeout.String())
			err = adapter.StopScan()
			if err != nil {
				println("failed to stop scan:", err.Error())
			}
		}
	})
}

func GetService(device *bluetooth.Device, serviceUUID bluetooth.UUID) (*bluetooth.DeviceService, error) {
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, errors.New("service not found:" + serviceUUID.String())
	}

	return &services[0], nil
}

func GetCharacteristics(device *bluetooth.Device, serviceUUID bluetooth.UUID, characteristicUUIDs ...bluetooth.UUID) ([]bluetooth.DeviceCharacteristic, error) {
	println("getting characteristics for service:", serviceUUID.String())
	service, err := GetService(device, serviceUUID)
	if err != nil {
		return nil, err
	}
	println("discovering characteristics:", characteristicUUIDs)
	return service.DiscoverCharacteristics(characteristicUUIDs)
}

func ReadCharacteristic(characteristic bluetooth.DeviceCharacteristic) ([]byte, error) {
	mtu, err := characteristic.GetMTU()
	if err != nil {
		mtu = defaultMTU
	}
	if mtu <= 0 {
		mtu = defaultMTU
	}
	buf := make([]byte, mtu)
	read, err := characteristic.Read(buf)
	if err != nil {
		return nil, err
	}
	println("read", read, "bytes from characteristic", buf[:read])
	return buf[:read], nil
}

func ReadCharacteristicAsString(characteristic bluetooth.DeviceCharacteristic) (string, error) {
	data, err := ReadCharacteristic(characteristic)
	if err != nil {
		return "", err
	}
	for _, d := range data {
		print("0x", d, ", ")
	}
	println()
	return string(data), nil
}
