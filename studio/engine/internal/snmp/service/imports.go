package service

import (
	"fmt"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/profiles"
	"studio-engine/internal/snmp/snmpfile"
	"studio-engine/internal/tools/pytext"
)

// maxImport is the largest walk or CSV text accepted.
const maxImport = 50 << 20

// Imported is a new device made from a file, with the lines that could not be used.
type Imported struct {
	Device   model.Device `json:"device"`
	Problems []string     `json:"problems"`
}

// ImportDevice makes a simulated device from snmpwalk output ("walk") or ScadaBench CSV
// ("csv"). Lines that cannot be used are listed; the device keeps the rest.
func (s *Service) ImportDevice(kind, name, text string) (Imported, error) {
	if len(text) > maxImport {
		return Imported{}, apperr.ValueError("The file is larger than 50 MB")
	}
	library := s.MIBs.Lib()
	var points []model.Point
	var problems []snmpfile.Problem
	var err error
	switch kind {
	case profiles.Walk:
		points, problems, err = snmpfile.ParseWalk(text, library)
	case profiles.CSV:
		points, problems, err = snmpfile.ParseCSV(text, library)
	default:
		return Imported{}, apperr.ValueError("Choose an snmpwalk file or a ScadaBench CSV file")
	}
	if err != nil {
		return Imported{}, apperr.ValueError(err.Error())
	}
	return s.addImported(kind, name, points, problems)
}

func (s *Service) addImported(kind, name string, points []model.Point, problems []snmpfile.Problem) (Imported, error) {
	config := s.Config()
	device := NewDevice(profiles.Empty, config.Devices)
	device.Profile, device.Points = kind, points
	if name = strings.TrimSpace(name); name != "" {
		device.Name = name
	} else {
		device.Name = strings.Replace(device.Name, profiles.Names[profiles.Empty], profiles.Names[kind], 1)
	}
	if err := checkPoints(device.Name, points); err != nil {
		return Imported{}, apperr.ValueError(err.Error())
	}
	config.Devices = append(config.Devices, device)
	if err := s.saveAndRestart(config); err != nil {
		return Imported{}, err
	}
	texts := make([]string, len(problems))
	for index, problem := range problems {
		texts[index] = fmt.Sprintf("Line %d: %s", problem.Line, problem.Text)
	}
	return Imported{Device: device, Problems: texts}, nil
}

// SaveReadDevice turns the last walk of the device being read into a simulated twin.
func (s *Service) SaveReadDevice(name string) (Imported, error) {
	s.mu.Lock()
	rows := make([]*pyjson.Object, 0, len(s.reader.order))
	for _, oid := range s.reader.order {
		rows = append(rows, s.reader.table[oid])
	}
	s.mu.Unlock()
	if len(rows) == 0 {
		return Imported{}, apperr.ValueError("Connect and walk a device first")
	}
	points := []model.Point{}
	for _, row := range rows {
		if row == nil {
			continue
		}
		kind := pytext.Str(field(row, "type"))
		if kind == "" {
			continue // exceptions such as noSuchInstance
		}
		hex, _ := field(row, "hex").(bool)
		point := model.Point{OID: pytext.Str(field(row, "oid")), Type: kind, Value: pytext.Str(field(row, "value")),
			Hex: hex, Behaviour: model.BehaviourFixed}
		if point.OID == "1.3.6.1.2.1.1.3.0" {
			point.Behaviour = model.BehaviourUptime
		}
		points = append(points, point)
	}
	return s.addImported(profiles.Walk, name, points, nil)
}

// ExportCSV is a device's points as ScadaBench CSV.
func (s *Service) ExportCSV(id string) (string, string, error) {
	device, err := deviceOf(s.Config(), id)
	if err != nil {
		return "", "", err
	}
	return snmpfile.WriteCSV(device.Points, s.MIBs.Lib()), device.Name, nil
}

// SavePoints replaces a device's points (the page's point editor and CSV import into a device).
func (s *Service) SavePoints(id string, points []model.Point) error {
	config := s.Config()
	index, err := config.findDevice(id)
	if err != nil {
		return apperr.ValueError(err.Error())
	}
	if points == nil {
		points = []model.Point{}
	}
	if err := checkPoints(config.Devices[index].Name, points); err != nil {
		return apperr.ValueError(err.Error())
	}
	config.Devices[index].Points = points
	return s.saveAndRestart(config)
}
