package service_test

import (
	"studio-engine/internal/api/toolsapi"
	gateway "studio-engine/internal/gateway/service"
	"studio-engine/internal/project/service"
	"studio-engine/internal/runtime"
)

// The project actor provides the ports the other services declare.
var (
	_ runtime.Project  = (*service.Service)(nil)
	_ gateway.Projects = (*service.Service)(nil)
	_ toolsapi.Projects = (*service.Service)(nil)
)
