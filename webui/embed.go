package webui

import "embed"

// Dist is the production SPA build copied in during the Docker image build.
//
//go:embed all:dist
var Dist embed.FS
