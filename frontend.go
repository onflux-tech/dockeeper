package dockeeper

import "embed"

// FrontendDist embeds the production React build output.
// Run `make build-web` before `go build` to ensure web/dist/ is populated.
//
//go:embed all:web/dist
var FrontendDist embed.FS
