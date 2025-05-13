package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/seanomeara96/auth"
)

func server(port string, mode Mode, service *Service) error {
	if port == "" {
		return fmt.Errorf("port is required via -port flag")
	}
	if mode == "" || mode == "dev" {
		log.Println("Starting server in development mode.")
		mode = Dev
	}
	productionDomain := os.Getenv("PROD_DOMAIN")
	if mode == Prod && productionDomain == "" {
		return errors.New("must supply production domain to run server in prod")
	}
	currentDomain := "http://localhost:" + port
	if mode == Prod {
		currentDomain = productionDomain
	}
	r := http.NewServeMux()

	tmpl := &Tmpl{
		mode: mode,
		glob: "templates/**/*.tmpl",
	}

	renderer := &Renderer{
		mode: mode,
		tmpl: tmpl,
	}

	authConfig := auth.AuthConfig{
		JWTSecretKey: os.Getenv("SESSION_KEY"),
		CookieSecure: true,
		HttpOnly:     true,
		SameSite:     http.SameSiteStrictMode,
	}
	authenticator, err := auth.Init(authConfig)
	if err != nil {
		return err
	}
	defer authenticator.Close()

	authenticator.Register(context.Background(), os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD"))

	handler := Handler{
		store:         sessions.NewCookieStore([]byte(os.Getenv(`SESSION_KEY`))),
		mode:          mode,
		domain:        currentDomain,
		service:       service,
		render:        renderer,
		authenticator: authenticator,
	}

	assetsDir := http.Dir("assets/dist")
	assetsFileServer := http.FileServer(assetsDir)
	r.Handle("/assets/", http.StripPrefix("/assets/", assetsFileServer))

	imageDir := http.Dir("static/website_screenshots")
	imagesFileServer := http.FileServer(imageDir)
	r.Handle("/website_screenshots/", http.StripPrefix("/website_screenshots/", imagesFileServer))

	faviconDir := http.Dir("favicon_io")
	faviconFileServer := http.FileServer(faviconDir)
	r.Handle("/favicon_io/", http.StripPrefix("/favicon_io/", faviconFileServer))

	/*
		Serve robots.txt & sitemap
	*/
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/robots.txt")
	})
	r.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/sitemap.xml")
	})
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./favicon_io/favicon.ico")
	})

	globalMiddleware := []middleware{handler.pathLogger}

	handle := newHandleFunc(r, globalMiddleware)

	handle("/", handler.handleGetHomePage)
	handle("GET /coupons", handler.handleListCoupons)
	handle("GET /website/{websitePath}", handler.handleGetFeed)
	handle("GET /subscribe", handler.handleSubscribe)
	handle("POST /subscribe", handler.handleStoreSubscription)
	handle("GET /subscribe/verify", handler.handleGetVerifySubscription)

	handle("GET /admin/signin", handler.adminHandleGetSignIn)
	handle("POST /admin/signin", handler.adminHandlePostSignIn)
	handle("GET /admin/signout", handler.adminHandleGetSignOut)
	handle("GET /admin", handler.mustBeAdmin(handler.adminHandleGetDashboard))

	handle("GET /admin/subscribers", handler.mustBeAdmin(handler.handleListSubscribers))
	/*	handle("GET /admin/subscribers/create", handler.mustBeAdmin(handler.handleCreateSubscriber))
		handle("POST /admin/subscribers/create", handler.mustBeAdmin(handler.handleStoreSubscriber))
		handle("GET /admin/subscribers/{id}", handler.mustBeAdmin(handler.handleEditSubscriber))
		handle("PUT /admin/subscribers/{id}", handler.mustBeAdmin(handler.handleUpdateSubscriber))
		handle("GET /admin/subscribers/delete/{id}", handler.mustBeAdmin(handler.handleDeleteSubscriberConfirmation))
		handle("DELETE /admin/subscribers/{id}", handler.mustBeAdmin(handler.handleDeleteSubscriber))*/

	/*
		Not part of the MVP
		handle("GET /admin/posts", handler.mustBeAdmin(handler.handleListPosts))
		handle("GET /admin/posts/{id}", handler.mustBeAdmin(handler.handleEditPost))
		handle("PUT /admin/posts/{id}", handler.mustBeAdmin(handler.handleUpdatePost))
		handle("GET /admin/posts/delete/{id}", handler.mustBeAdmin(handler.handleDeletePostConfirmation))
		handle("DELETE /admin/posts/{id}", handler.mustBeAdmin(handler.handleDeletePost))

		handle("GET /admin/brands", handler.mustBeAdmin(handler.handleListBrands))
		handle("GET /admin/brands/create", handler.mustBeAdmin(handler.handleCreateBrand))
		handle("POST /admin/brands/create", handler.mustBeAdmin(handler.handleStoreBrand))
		handle("GET /admin/brands/{id}", handler.mustBeAdmin(handler.handleEditBrand))
		handle("PUT /admin/brands/{id}", handler.mustBeAdmin(handler.handleUpdateBrand))
		handle("PATCH /admin/brands/{id}/score", handler.mustBeAdmin(handler.handleUpdateBrandScore))
		handle("GET /admin/brands/delete/{id}", handler.mustBeAdmin(handler.handleDeleteBrandConfirmation))
		handle("DELETE /admin/brands/{id}", handler.mustBeAdmin(handler.handleDeleteBrand))

		handle("GET /admin/categories", handler.mustBeAdmin(handler.handleListCategories))
		handle("GET /admin/categories/create", handler.mustBeAdmin(handler.handleCreateCategory))
		handle("POST /admin/categories/create", handler.mustBeAdmin(handler.handleStoreCategory))
		handle("GET /admin/categories/{id}", handler.mustBeAdmin(handler.handleEditCategory))
		handle("PUT /admin/categories/{id}", handler.mustBeAdmin(handler.handleUpdateCategory))
		handle("GET /admin/categories/delete/{id}", handler.mustBeAdmin(handler.handleDeleteCategoryConfirmation))
		handle("DELETE /admin/categories/{id}", handler.mustBeAdmin(handler.handleDeleteCategory))


	*/

	log.Println("Server listening on http://localhost:" + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("failure to launch. %w", err)
	}

	return nil
}
