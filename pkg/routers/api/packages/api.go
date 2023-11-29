// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package packages

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gitbundle/modules/setting"
	auth "github.com/gitbundle/server/pkg/auth/manager"
	"github.com/gitbundle/server/pkg/context"
	context_service "github.com/gitbundle/server/pkg/context"
	"github.com/gitbundle/server/pkg/perm"
	"github.com/gitbundle/server/pkg/routers/api/packages/composer"
	"github.com/gitbundle/server/pkg/routers/api/packages/conan"
	"github.com/gitbundle/server/pkg/routers/api/packages/container"
	"github.com/gitbundle/server/pkg/routers/api/packages/generic"
	"github.com/gitbundle/server/pkg/routers/api/packages/helm"
	"github.com/gitbundle/server/pkg/routers/api/packages/maven"
	"github.com/gitbundle/server/pkg/routers/api/packages/npm"
	"github.com/gitbundle/server/pkg/routers/api/packages/nuget"
	"github.com/gitbundle/server/pkg/routers/api/packages/pypi"
	"github.com/gitbundle/server/pkg/routers/api/packages/rubygems"
	"github.com/gitbundle/server/pkg/static/templates"
	"github.com/gitbundle/server/pkg/web"
)

func reqPackageAccess(accessMode perm.AccessMode) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		if ctx.Package.AccessMode < accessMode && !ctx.IsUserSiteAdmin() {
			ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="GitBundle Package API"`)
			ctx.Error(http.StatusUnauthorized, "reqPackageAccess", "user should have specific permission or be a site admin")
			return
		}
	}
}

func Routes() *web.Route {
	r := web.NewRoute()

	r.Use(templates.PackageContexter())

	authMethods := []auth.Method{
		&auth.OAuth2{},
		&auth.Basic{},
		&conan.Auth{},
	}
	if setting.Service.EnableReverseProxyAuth {
		authMethods = append(authMethods, &auth.ReverseProxy{})
	}

	authGroup := auth.NewGroup(authMethods...)
	r.Use(func(ctx *context.Context) {
		ctx.Doer = authGroup.Verify(ctx.Req, ctx.Resp, ctx, ctx.Session)
	})

	r.Group("/{username}", func() {
		r.Group("/composer", func() {
			r.Get("/packages.json", composer.ServiceIndex)
			r.Get("/search.json", composer.SearchPackages)
			r.Get("/list.json", composer.EnumeratePackages)
			r.Get("/p2/{vendorname}/{projectname}~dev.json", composer.PackageMetadata)
			r.Get("/p2/{vendorname}/{projectname}.json", composer.PackageMetadata)
			r.Get("/files/{package}/{version}/{filename}", composer.DownloadPackageFile)
			r.Put("", reqPackageAccess(perm.AccessModeWrite), composer.UploadPackage)
		})
		r.Group("/conan", func() {
			r.Group("/v1", func() {
				r.Get("/ping", conan.Ping)
				r.Group("/users", func() {
					r.Get("/authenticate", conan.Authenticate)
					r.Get("/check_credentials", conan.CheckCredentials)
				})
				r.Group("/conans", func() {
					r.Get("/search", conan.SearchRecipes)
					r.Group("/{name}/{version}/{user}/{channel}", func() {
						r.Get("", conan.RecipeSnapshot)
						r.Delete("", reqPackageAccess(perm.AccessModeWrite), conan.DeleteRecipeV1)
						r.Get("/search", conan.SearchPackagesV1)
						r.Get("/digest", conan.RecipeDownloadURLs)
						r.Post("/upload_urls", reqPackageAccess(perm.AccessModeWrite), conan.RecipeUploadURLs)
						r.Get("/download_urls", conan.RecipeDownloadURLs)
						r.Group("/packages", func() {
							r.Post("/delete", reqPackageAccess(perm.AccessModeWrite), conan.DeletePackageV1)
							r.Group("/{package_reference}", func() {
								r.Get("", conan.PackageSnapshot)
								r.Get("/digest", conan.PackageDownloadURLs)
								r.Post("/upload_urls", reqPackageAccess(perm.AccessModeWrite), conan.PackageUploadURLs)
								r.Get("/download_urls", conan.PackageDownloadURLs)
							})
						})
					}, conan.ExtractPathParameters)
				})
				r.Group("/files/{name}/{version}/{user}/{channel}/{recipe_revision}", func() {
					r.Group("/recipe/{filename}", func() {
						r.Get("", conan.DownloadRecipeFile)
						r.Put("", reqPackageAccess(perm.AccessModeWrite), conan.UploadRecipeFile)
					})
					r.Group("/package/{package_reference}/{package_revision}/{filename}", func() {
						r.Get("", conan.DownloadPackageFile)
						r.Put("", reqPackageAccess(perm.AccessModeWrite), conan.UploadPackageFile)
					})
				}, conan.ExtractPathParameters)
			})
			r.Group("/v2", func() {
				r.Get("/ping", conan.Ping)
				r.Group("/users", func() {
					r.Get("/authenticate", conan.Authenticate)
					r.Get("/check_credentials", conan.CheckCredentials)
				})
				r.Group("/conans", func() {
					r.Get("/search", conan.SearchRecipes)
					r.Group("/{name}/{version}/{user}/{channel}", func() {
						r.Delete("", reqPackageAccess(perm.AccessModeWrite), conan.DeleteRecipeV2)
						r.Get("/search", conan.SearchPackagesV2)
						r.Get("/latest", conan.LatestRecipeRevision)
						r.Group("/revisions", func() {
							r.Get("", conan.ListRecipeRevisions)
							r.Group("/{recipe_revision}", func() {
								r.Delete("", reqPackageAccess(perm.AccessModeWrite), conan.DeleteRecipeV2)
								r.Get("/search", conan.SearchPackagesV2)
								r.Group("/files", func() {
									r.Get("", conan.ListRecipeRevisionFiles)
									r.Group("/{filename}", func() {
										r.Get("", conan.DownloadRecipeFile)
										r.Put("", reqPackageAccess(perm.AccessModeWrite), conan.UploadRecipeFile)
									})
								})
								r.Group("/packages", func() {
									r.Delete("", reqPackageAccess(perm.AccessModeWrite), conan.DeletePackageV2)
									r.Group("/{package_reference}", func() {
										r.Delete("", reqPackageAccess(perm.AccessModeWrite), conan.DeletePackageV2)
										r.Get("/latest", conan.LatestPackageRevision)
										r.Group("/revisions", func() {
											r.Get("", conan.ListPackageRevisions)
											r.Group("/{package_revision}", func() {
												r.Delete("", reqPackageAccess(perm.AccessModeWrite), conan.DeletePackageV2)
												r.Group("/files", func() {
													r.Get("", conan.ListPackageRevisionFiles)
													r.Group("/{filename}", func() {
														r.Get("", conan.DownloadPackageFile)
														r.Put("", reqPackageAccess(perm.AccessModeWrite), conan.UploadPackageFile)
													})
												})
											})
										})
									})
								})
							})
						})
					}, conan.ExtractPathParameters)
				})
			})
		})
		r.Group("/generic", func() {
			r.Group("/{packagename}/{packageversion}/{filename}", func() {
				r.Get("", generic.DownloadPackageFile)
				r.Group("", func() {
					r.Put("", generic.UploadPackage)
					r.Delete("", generic.DeletePackage)
				}, reqPackageAccess(perm.AccessModeWrite))
			})
		})
		r.Group("/helm", func() {
			r.Get("/index.yaml", helm.Index)
			r.Get("/{filename}", helm.DownloadPackageFile)
			r.Post("/api/charts", reqPackageAccess(perm.AccessModeWrite), helm.UploadPackage)
		})
		r.Group("/maven", func() {
			r.Put("/*", reqPackageAccess(perm.AccessModeWrite), maven.UploadPackageFile)
			r.Get("/*", maven.DownloadPackageFile)
		})
		r.Group("/nuget", func() {
			r.Get("/index.json", nuget.ServiceIndex)
			r.Get("/query", nuget.SearchService)
			r.Group("/registration/{id}", func() {
				r.Get("/index.json", nuget.RegistrationIndex)
				r.Get("/{version}", nuget.RegistrationLeaf)
			})
			r.Group("/package/{id}", func() {
				r.Get("/index.json", nuget.EnumeratePackageVersions)
				r.Get("/{version}/{filename}", nuget.DownloadPackageFile)
			})
			r.Group("", func() {
				r.Put("/", nuget.UploadPackage)
				r.Put("/symbolpackage", nuget.UploadSymbolPackage)
				r.Delete("/{id}/{version}", nuget.DeletePackage)
			}, reqPackageAccess(perm.AccessModeWrite))
			r.Get("/symbols/{filename}/{guid:[0-9a-f]{32}}FFFFFFFF/{filename2}", nuget.DownloadSymbolFile)
		})
		r.Group("/npm", func() {
			r.Group("/@{scope}/{id}", func() {
				r.Get("", npm.PackageMetadata)
				r.Put("", reqPackageAccess(perm.AccessModeWrite), npm.UploadPackage)
				r.Get("/-/{version}/{filename}", npm.DownloadPackageFile)
			})
			r.Group("/{id}", func() {
				r.Get("", npm.PackageMetadata)
				r.Put("", reqPackageAccess(perm.AccessModeWrite), npm.UploadPackage)
				r.Get("/-/{version}/{filename}", npm.DownloadPackageFile)
			})
			r.Group("/-/package/@{scope}/{id}/dist-tags", func() {
				r.Get("", npm.ListPackageTags)
				r.Group("/{tag}", func() {
					r.Put("", npm.AddPackageTag)
					r.Delete("", npm.DeletePackageTag)
				}, reqPackageAccess(perm.AccessModeWrite))
			})
			r.Group("/-/package/{id}/dist-tags", func() {
				r.Get("", npm.ListPackageTags)
				r.Group("/{tag}", func() {
					r.Put("", npm.AddPackageTag)
					r.Delete("", npm.DeletePackageTag)
				}, reqPackageAccess(perm.AccessModeWrite))
			})
		})
		r.Group("/pypi", func() {
			r.Post("/", reqPackageAccess(perm.AccessModeWrite), pypi.UploadPackageFile)
			r.Get("/files/{id}/{version}/{filename}", pypi.DownloadPackageFile)
			r.Get("/simple/{id}", pypi.PackageMetadata)
		})
		r.Group("/rubygems", func() {
			r.Get("/specs.4.8.gz", rubygems.EnumeratePackages)
			r.Get("/latest_specs.4.8.gz", rubygems.EnumeratePackagesLatest)
			r.Get("/prerelease_specs.4.8.gz", rubygems.EnumeratePackagesPreRelease)
			r.Get("/quick/Marshal.4.8/{filename}", rubygems.ServePackageSpecification)
			r.Get("/gems/{filename}", rubygems.DownloadPackageFile)
			r.Group("/api/v1/gems", func() {
				r.Post("/", rubygems.UploadPackageFile)
				r.Delete("/yank", rubygems.DeletePackage)
			}, reqPackageAccess(perm.AccessModeWrite))
		})
	}, context_service.UserAssignmentWeb(), context.PackageAssignment(), reqPackageAccess(perm.AccessModeRead))

	return r
}

func ContainerRoutes() *web.Route {
	r := web.NewRoute()

	r.Use(templates.PackageContexter())

	authMethods := []auth.Method{
		&auth.Basic{},
		&container.Auth{},
	}
	if setting.Service.EnableReverseProxyAuth {
		authMethods = append(authMethods, &auth.ReverseProxy{})
	}

	authGroup := auth.NewGroup(authMethods...)
	r.Use(func(ctx *context.Context) {
		ctx.Doer = authGroup.Verify(ctx.Req, ctx.Resp, ctx, ctx.Session)
	})

	r.Get("", container.ReqContainerAccess, container.DetermineSupport)
	r.Get("/token", container.Authenticate)
	r.Get("/_catalog", container.ReqContainerAccess, container.GetRepositoryList)
	r.Group("/{username}", func() {
		r.Group("/{image}", func() {
			r.Group("/blobs/uploads", func() {
				r.Post("", container.InitiateUploadBlob)
				r.Group("/{uuid}", func() {
					r.Patch("", container.UploadBlob)
					r.Put("", container.EndUploadBlob)
				})
			}, reqPackageAccess(perm.AccessModeWrite))
			r.Group("/blobs/{digest}", func() {
				r.Head("", container.HeadBlob)
				r.Get("", container.GetBlob)
				r.Delete("", reqPackageAccess(perm.AccessModeWrite), container.DeleteBlob)
			})
			r.Group("/manifests/{reference}", func() {
				r.Put("", reqPackageAccess(perm.AccessModeWrite), container.UploadManifest)
				r.Head("", container.HeadManifest)
				r.Get("", container.GetManifest)
				r.Delete("", reqPackageAccess(perm.AccessModeWrite), container.DeleteManifest)
			})
			r.Get("/tags/list", container.GetTagList)
		}, container.VerifyImageName)

		var (
			blobsUploadsPattern = regexp.MustCompile(`\A(.+)/blobs/uploads/([a-zA-Z0-9-_.=]+)\z`)
			blobsPattern        = regexp.MustCompile(`\A(.+)/blobs/([^/]+)\z`)
			manifestsPattern    = regexp.MustCompile(`\A(.+)/manifests/([^/]+)\z`)
		)

		// Manual mapping of routes because {image} can contain slashes which chi does not support
		r.Route("/*", "HEAD,GET,POST,PUT,PATCH,DELETE", func(ctx *context.Context) {
			path := ctx.Params("*")
			isHead := ctx.Req.Method == "HEAD"
			isGet := ctx.Req.Method == "GET"
			isPost := ctx.Req.Method == "POST"
			isPut := ctx.Req.Method == "PUT"
			isPatch := ctx.Req.Method == "PATCH"
			isDelete := ctx.Req.Method == "DELETE"

			if isPost && strings.HasSuffix(path, "/blobs/uploads") {
				reqPackageAccess(perm.AccessModeWrite)(ctx)
				if ctx.Written() {
					return
				}

				ctx.SetParams("image", path[:len(path)-14])
				container.VerifyImageName(ctx)
				if ctx.Written() {
					return
				}

				container.InitiateUploadBlob(ctx)
				return
			}
			if isGet && strings.HasSuffix(path, "/tags/list") {
				ctx.SetParams("image", path[:len(path)-10])
				container.VerifyImageName(ctx)
				if ctx.Written() {
					return
				}

				container.GetTagList(ctx)
				return
			}

			m := blobsUploadsPattern.FindStringSubmatch(path)
			if len(m) == 3 && (isPut || isPatch) {
				reqPackageAccess(perm.AccessModeWrite)(ctx)
				if ctx.Written() {
					return
				}

				ctx.SetParams("image", m[1])
				container.VerifyImageName(ctx)
				if ctx.Written() {
					return
				}

				ctx.SetParams("uuid", m[2])

				if isPatch {
					container.UploadBlob(ctx)
				} else {
					container.EndUploadBlob(ctx)
				}
				return
			}
			m = blobsPattern.FindStringSubmatch(path)
			if len(m) == 3 && (isHead || isGet || isDelete) {
				ctx.SetParams("image", m[1])
				container.VerifyImageName(ctx)
				if ctx.Written() {
					return
				}

				ctx.SetParams("digest", m[2])

				if isHead {
					container.HeadBlob(ctx)
				} else if isGet {
					container.GetBlob(ctx)
				} else {
					reqPackageAccess(perm.AccessModeWrite)(ctx)
					if ctx.Written() {
						return
					}
					container.DeleteBlob(ctx)
				}
				return
			}
			m = manifestsPattern.FindStringSubmatch(path)
			if len(m) == 3 && (isHead || isGet || isPut || isDelete) {
				ctx.SetParams("image", m[1])
				container.VerifyImageName(ctx)
				if ctx.Written() {
					return
				}

				ctx.SetParams("reference", m[2])

				if isHead {
					container.HeadManifest(ctx)
				} else if isGet {
					container.GetManifest(ctx)
				} else {
					reqPackageAccess(perm.AccessModeWrite)(ctx)
					if ctx.Written() {
						return
					}
					if isPut {
						container.UploadManifest(ctx)
					} else {
						container.DeleteManifest(ctx)
					}
				}
				return
			}

			ctx.Status(http.StatusNotFound)
		})
	}, container.ReqContainerAccess, context_service.UserAssignmentWeb(), context.PackageAssignment(), reqPackageAccess(perm.AccessModeRead))

	return r
}
