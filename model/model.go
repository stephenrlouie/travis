/* Object model for Controller Core.

To be used for: database model wrapper, REST API, and job scheduler components.

Plugin: description for service operations
Site: one micro data center. a group of Nodes.
Jobs: JSON-defined tasks able to run a container job(s) on one or more Sites or Nodes.


*/

package model

import (
	"time"

	"github.com/pborman/uuid"
)

func GenerateUUID() string {
	return uuid.New()
}

type Metadataer interface {
	Set(string, interface{})
	Get(string) (interface{}, bool)
	GetString(string) (string, bool)
}

// Metadata for persistent storage
type MetaImpl struct {
	Metadata map[string]interface{}
}

func (m *MetaImpl) Set(k string, v interface{}) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[k] = v
}

func (m *MetaImpl) Get(k string) (interface{}, bool) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	v, ok := m.Metadata[k]
	return v, ok
}

func (m *MetaImpl) GetString(k string) (string, bool) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	if v, ok := m.Metadata[k]; ok {
		if v2, ok2 := v.(string); ok2 {
			return v2, ok2
		}
	}
	return "", false
}

type Plugin struct {
	MetaImpl

	Id   string
	Name string

	Summary string

	Description string

	Maintainer string

	Version string

	Image string

	Input []PluginDataItem

	Output []PluginDataItem

	Config map[string]interface{}
}

type PluginDataItem struct {
	Name string

	Type string

	// Default either a single string or list of strings. Empty if none specified
	Default []string
}

const (
	ServiceStatusActive   = "Active"
	ServiceStatusInactive = "Inactive"
	ServiceStatusStopped  = "Stopped"
	ServiceStatusFailed   = "Failed"
)

type Service struct {
	MetaImpl

	Id   string
	Name string

	Enabled bool

	Plugin string

	Status string

	Updated time.Time

	Input []ServiceDataItem

	Output []ServiceDataItem
}

func ServiceDataItemsToMap(in []ServiceDataItem) map[string][]string {
	sdimap := make(map[string][]string)
	for _, i := range in {
		sdimap[i.Name] = i.Value
	}
	return sdimap
}

type ServiceDataItem struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Value []string `json:"value"`

	From []string `json:"from"`

	// Set to true if the value(s) have been set by an operation
	Set bool `json:"set"`
}

const (
	ServiceOperationDeploy  = "Deploy"
	ServiceOperationStatus  = "Status"
	ServiceOperationUpdate  = "Update"
	ServiceOperationDestroy = "Destroy"
)

const (
	ServiceOperationStatusWaiting  = "Inactive"
	ServiceOperationStatusRunning  = "Running"
	ServiceOperationStatusFinished = "Finished"
	ServiceOperationStatusFailed   = "Error"
	ServiceOperationStatusStopped  = "Stopped"
	ServiceOperationStatusUnknown  = "Unknown"
)

// Only one Operation should be active at a time for a given Service
type ServiceOperation struct {
	MetaImpl

	Operation string

	Status string

	Created time.Time

	Started time.Time

	Plugin Plugin

	Service Service
}
