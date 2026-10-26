//go:build windows

package labnet

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// netClass is GUID_DEVCLASS_NET.
var netClass = windows.GUID{Data1: 0x4d36e972, Data2: 0xe325, Data3: 0x11ce,
	Data4: [8]byte{0xbf, 0xc1, 0x08, 0x00, 0x2b, 0xe1, 0x03, 0x18}}

const loopbackHardwareID = "*MSLOOP"

var (
	newdev                                = windows.NewLazySystemDLL("newdev.dll")
	procUpdateDriverForPlugAndPlayDevices = newdev.NewProc("UpdateDriverForPlugAndPlayDevicesW")
	procDiUninstallDevice                 = newdev.NewProc("DiUninstallDevice")
)

// CreateCard creates one Windows loopback card from the driver Windows ships (netloop.inf),
// as Device Manager's "Add legacy hardware" does, and returns its adapter GUID once Windows
// lists it. Needs administrator rights.
func CreateCard() (string, error) {
	before := map[string]bool{}
	if adapters, err := ListAdapters(); err == nil {
		for _, adapter := range adapters {
			before[adapter.GUID] = true
		}
	}
	set, err := windows.SetupDiCreateDeviceInfoListEx(&netClass, 0, "")
	if err != nil {
		return "", err
	}
	defer set.Close()
	data, err := set.CreateDeviceInfo("NET", &netClass, "", 0, windows.DICD_GENERATE_ID)
	if err != nil {
		return "", err
	}
	if err := set.SetDeviceRegistryProperty(data, windows.SPDRP_HARDWAREID, multiString(loopbackHardwareID)); err != nil {
		return "", err
	}
	if err := set.CallClassInstaller(windows.DIF_REGISTERDEVICE, data); err != nil {
		return "", err
	}
	inf, _ := windows.UTF16PtrFromString(filepath.Join(os.Getenv("WINDIR"), "INF", "netloop.inf"))
	id, _ := windows.UTF16PtrFromString(loopbackHardwareID)
	var reboot int32
	const installForce = 1
	if ok, _, err := procUpdateDriverForPlugAndPlayDevices.Call(0, uintptr(unsafe.Pointer(id)),
		uintptr(unsafe.Pointer(inf)), installForce, uintptr(unsafe.Pointer(&reboot))); ok == 0 {
		set.CallClassInstaller(windows.DIF_REMOVE, data)
		return "", err
	}
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		adapters, _ := ListAdapters()
		for _, adapter := range adapters {
			if !before[adapter.GUID] && IsCard(adapter.Description) {
				return adapter.GUID, nil
			}
		}
	}
	return "", errors.New("Windows created the loopback card but did not list it within 20 seconds")
}

// RemoveCard uninstalls the loopback card whose adapter GUID is guid. Only a KM-TEST card is
// ever removed.
func RemoveCard(guid string) error {
	set, err := windows.SetupDiGetClassDevsEx(&netClass, "", 0, windows.DIGCF_PRESENT, 0, "")
	if err != nil {
		return err
	}
	defer set.Close()
	for index := 0; ; index++ {
		data, err := set.EnumDeviceInfo(index)
		if errors.Is(err, windows.ERROR_NO_MORE_ITEMS) {
			return nil // already gone
		}
		if err != nil {
			continue
		}
		if !strings.EqualFold(instanceGUID(set, data), guid) {
			continue
		}
		if ids, _ := set.DeviceRegistryProperty(data, windows.SPDRP_HARDWAREID); !hasLoopbackID(ids) {
			return errors.New("that adapter is not a Windows loopback card; it was not removed")
		}
		var reboot int32
		if ok, _, err := procDiUninstallDevice.Call(0, uintptr(set), uintptr(unsafe.Pointer(data)), 0,
			uintptr(unsafe.Pointer(&reboot))); ok == 0 {
			return err
		}
		return nil
	}
}

func instanceGUID(set windows.DevInfo, data *windows.DevInfoData) string {
	handle, err := set.OpenDevRegKey(data, windows.DICS_FLAG_GLOBAL, 0, windows.DIREG_DRV, windows.KEY_READ)
	if err != nil {
		return ""
	}
	key := registry.Key(handle)
	defer key.Close()
	value, _, _ := key.GetStringValue("NetCfgInstanceId")
	return value
}

func hasLoopbackID(value any) bool {
	ids, _ := value.([]string)
	for _, id := range ids {
		if strings.EqualFold(id, loopbackHardwareID) {
			return true
		}
	}
	return false
}

// multiString is a REG_MULTI_SZ value with one string: UTF-16, then two NULs.
func multiString(text string) []byte {
	units := append(utf16.Encode([]rune(text)), 0, 0)
	raw := make([]byte, 0, len(units)*2)
	for _, unit := range units {
		raw = append(raw, byte(unit), byte(unit>>8))
	}
	return raw
}
