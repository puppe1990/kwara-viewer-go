package app

import (
	"github.com/puppe1990/amarra-cais/pkg/amarra/live"
	"github.com/puppe1990/amarra-cais/pkg/cais"

	"github.com/puppe1990/kwara-viewer/internal/handlers"
)

func registerRoutes(r *cais.Router, deps Deps, cfg cais.Config) {
	catalog := handlers.NewCatalogHandler(deps.Views, deps.Store, deps.Searcher, deps.Site, deps.Catalog, cfg)
	r.Get("/", catalog.Home)
	r.Post("/sync", catalog.Sync)
	r.Get("/robots.txt", catalog.Robots)
}

func registerLiveViews(hub *live.Hub, deps Deps) {
	_ = hub
	_ = deps
	// cais:live-views
}
